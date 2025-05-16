// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	fedv1alpha1 "github.com/external-secrets/external-secrets/apis/federation/v1alpha1"
	store "github.com/external-secrets/external-secrets/pkg/federation/store"
	"github.com/labstack/echo/v4"
)

type ParseRSAPublicKeyTestSuite struct {
	suite.Suite
}

func (s *ParseRSAPublicKeyTestSuite) TestParseRSAPublicKey() {
	tests := []struct {
		name    string
		key     map[string]string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid RSA public key",
			key: map[string]string{
				// Standard RSA modulus (n) and exponent (e) values for testing
				// These values represent a valid but test-only RSA key
				"n":   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
				"e":   "AQAB",
				"alg": "RS256",
			},
			wantErr: false,
		},
		{
			name: "missing n value",
			key: map[string]string{
				"e":   "AQAB",
				"alg": "RS256",
			},
			wantErr: true,
			errMsg:  "n not found in key",
		},
		{
			name: "invalid n value - cannot be decoded",
			key: map[string]string{
				"n":   "XXXinvalid//?lid-base64-url",
				"e":   "smth",
				"alg": "RS256",
			},
			wantErr: true,
			errMsg:  "failed to decode modulus",
		},
		{
			name: "missing e value",
			key: map[string]string{
				"n":   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
				"alg": "RS256",
			},
			wantErr: true,
			errMsg:  "e not found in key",
		},
		{
			name: "invalid e value - cannot be decoded",
			key: map[string]string{
				"n":   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
				"e":   "XXXinvalid//?lid-base64-url",
				"alg": "RS256",
			},
			wantErr: true,
			errMsg:  "failed to decode exponent",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			key, err := parseRSAPublicKey(tt.key)
			if tt.wantErr {
				assert.Error(s.T(), err)
				if tt.errMsg != "" {
					assert.Contains(s.T(), err.Error(), tt.errMsg)
				}
				assert.Nil(s.T(), key)
			} else {
				assert.NoError(s.T(), err)
				assert.NotNil(s.T(), key)
			}
		})
	}
}

func TestParseRSAPublicKeyTestSuite(t *testing.T) {
	suite.Run(t, new(ParseRSAPublicKeyTestSuite))
}

// mockFederationProvider implements the FederationProvider interface for testing.
type mockFederationProvider struct {
	jwks map[string]map[string]string
	err  error
}

// GetJWKS implements the FederationProvider interface.
func (m *mockFederationProvider) GetJWKS(ctx context.Context, token, issuer string, caCrt []byte) (map[string]map[string]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.jwks, nil
}

type FindJWKSTestSuite struct {
	suite.Suite
	// Store specs to clean up after tests
	specs []*fedv1alpha1.AuthorizationSpec
}

func (s *FindJWKSTestSuite) SetupTest() {
	// Initialize specs slice
	s.specs = []*fedv1alpha1.AuthorizationSpec{}
}

func (s *FindJWKSTestSuite) TearDownTest() {
	// Clean up any specs added to the store
	for _, spec := range s.specs {
		store.Remove("test-issuer", spec)
	}
}

func (s *FindJWKSTestSuite) TestFindJWKS() {
	tests := []struct {
		name      string
		issuer    string
		onlyToken string
		caCrt     string
		setup     func() // Function to set up the test case
		expect    map[*fedv1alpha1.AuthorizationSpec]map[string]map[string]string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "successful JWKS retrieval",
			issuer:    "test-issuer",
			onlyToken: "test-token",
			caCrt:     "test-ca-cert",
			setup: func() {
				// Create a test authorization spec
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer",
					},
				}

				// Add the spec to the store
				store.Add("test-issuer", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return JWKS
				provider := &mockFederationProvider{
					jwks: map[string]map[string]string{
						"kid1": {
							"n":   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
							"e":   "AQAB",
							"alg": "RS256",
						},
					},
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)
			},
			// We don't need to pre-compute the expected result since we'll compare with the actual result
			// in the test case run function
			expect:  nil,
			wantErr: false,
		},
		{
			name:      "provider returns error",
			issuer:    "test-issuer-error",
			onlyToken: "test-token",
			caCrt:     "test-ca-cert",
			setup: func() {
				// Create a test authorization spec
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation-error",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer-error",
					},
				}

				// Add the spec to the store
				store.Add("test-issuer-error", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return an error
				provider := &mockFederationProvider{
					err: errors.New("provider error"),
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)
			},
			expect:  nil,
			wantErr: true,
			errMsg:  "no jwks found",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup the test
			if tt.setup != nil {
				tt.setup()
			}

			// Call the function
			result, err := findJWKS(context.Background(), tt.issuer, tt.onlyToken, tt.caCrt)

			// Check results
			if tt.wantErr {
				s.Error(err)
				if tt.errMsg != "" {
					s.Contains(err.Error(), tt.errMsg)
				}
			} else {
				s.NoError(err)

				// For the happy path, just check that the result is not nil
				// and contains the expected structure
				if tt.expect != nil {
					s.Equal(tt.expect, result)
				} else {
					s.NotNil(result, "Result should not be nil")

					// Check that the result contains at least one spec
					s.Greater(len(result), 0, "Result should contain at least one spec")

					// For each spec in the result, check that it has JWKS data
					for spec, jwksData := range result {
						s.NotNil(spec, "Spec should not be nil")
						s.NotNil(jwksData, "JWKS data should not be nil")
						s.Greater(len(jwksData), 0, "JWKS data should contain at least one key")

						// Check the first key in the JWKS data
						for kid, keyData := range jwksData {
							s.NotEmpty(kid, "Key ID should not be empty")
							s.NotNil(keyData, "Key data should not be nil")

							// Check that the key data contains the required fields
							s.Contains(keyData, "n", "Key data should contain modulus")
							s.Contains(keyData, "e", "Key data should contain exponent")
							s.Contains(keyData, "alg", "Key data should contain algorithm")
							break // Only check the first key
						}
						break // Only check the first spec
					}
				}
			}
		})
	}
}

func TestFindJWKSTestSuite(t *testing.T) {
	suite.Run(t, new(FindJWKSTestSuite))
}

type GenParseTokenTestSuite struct {
	suite.Suite
	server *ServerHandler
	specs  []*fedv1alpha1.AuthorizationSpec
}

func (s *GenParseTokenTestSuite) SetupTest() {
	// Initialize the server handler
	s.server = NewServerHandler(nil, ":8080")

	// Initialize the specMap
	s.server.specMap = make(map[string][]*fedv1alpha1.AuthorizationSpec)

	// Initialize specs slice for cleanup
	s.specs = []*fedv1alpha1.AuthorizationSpec{}
}

func (s *GenParseTokenTestSuite) TearDownTest() {
	// Clean up any specs added to the store
	for _, spec := range s.specs {
		store.Remove("test-issuer", spec)
	}
}

func (s *GenParseTokenTestSuite) TestGenParseToken() {
	tests := []struct {
		name      string
		onlyToken string
		caCrt     []byte
		setup     func() // Function to set up the test case
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "successful key retrieval",
			onlyToken: "test-token",
			caCrt:     []byte("test-ca-cert"),
			setup: func() {
				// Create a test authorization spec
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer",
					},
				}

				// Add the spec to the store
				store.Add("test-issuer", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return JWKS
				provider := &mockFederationProvider{
					jwks: map[string]map[string]string{
						"kid1": {
							"n":   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
							"e":   "AQAB",
							"alg": "RS256",
						},
					},
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)

				// Directly add the spec to the server's specMap to bypass the findJWKS call
				s.server.specMap["test-ca-cert"] = []*fedv1alpha1.AuthorizationSpec{spec}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup the test
			if tt.setup != nil {
				tt.setup()
			}

			// Create a signed test token with the required claims
			// In a real test, we would sign this with a private key
			// For our test, we'll use a mock token
			const mockValidJWT = "eyJhbGciOiJSUzI1NiIsImtpZCI6ImtpZDEifQ.eyJpc3MiOiJ0ZXN0LWlzc3VlciIsInN1YiI6InRlc3Qtc3ViamVjdCJ9.dGhpc19pc19hX3ZhbGlkX2Jhc2U2NF9lbmNvZGVkX3NpZ25hdHVyZQ"
			mockToken := mockValidJWT

			// Add this to the test case run function, before calling genParseToken
			s.T().Logf("Before genParseToken, s.server.specMap = %+v", s.server.specMap)

			// Get the key parsing function
			keyFunc := s.server.genParseToken(context.Background(), mockToken, tt.caCrt)

			// Add this after calling genParseToken
			s.T().Logf("After genParseToken, s.server.specMap = %+v", s.server.specMap)
			// Create a token with the header and claims parts from our mock token
			// This simulates what jwt.Parse would do internally
			parts := strings.Split(mockToken, ".")
			if len(parts) != 3 {
				s.Fail("Invalid mock token")
			}

			// Create a token with the header and claims
			token := &jwt.Token{
				Raw: mockToken,
				Header: map[string]interface{}{
					"alg": "RS256",
					"kid": "kid1",
				},
				Claims: jwt.MapClaims{
					"iss": "test-issuer",
					"sub": "test-subject",
				},
				Signature: []byte(parts[2]),
				Method:    jwt.SigningMethodRS256,
			}

			// Call the key parsing function with our test token
			key, err := keyFunc(token)

			// Check results
			if tt.wantErr {
				s.Require().Error(err)
				s.T().Logf("Error: %v", err)
				if tt.errMsg != "" {
					s.Contains(err.Error(), tt.errMsg)
				}
				s.Nil(key)
			} else {
				s.NoError(err)
				s.NotNil(key)
			}
		})
	}
}

func TestGenParseTokenTestSuite(t *testing.T) {
	suite.Run(t, new(GenParseTokenTestSuite))
}

type ProcessRequestTestSuite struct {
	suite.Suite
	server *ServerHandler
	specs  []*fedv1alpha1.AuthorizationSpec
}

func (s *ProcessRequestTestSuite) SetupTest() {
	// Initialize the server handler
	s.server = NewServerHandler(nil, ":8080")

	// Initialize specs slice for cleanup
	s.specs = []*fedv1alpha1.AuthorizationSpec{}
}

func (s *ProcessRequestTestSuite) TearDownTest() {
	// Clean up any specs added to the store
	for _, spec := range s.specs {
		store.Remove("test-issuer", spec)
	}
}

func (s *ProcessRequestTestSuite) TestProcessRequest() {
	// Define test cases
	tests := []struct {
		name       string
		setup      func() *echo.Context
		wantIssuer string
		wantSub    string
		wantErr    bool
		errMsg     string
	}{
		{
			name: "missing authorization header",
			setup: func() *echo.Context {
				// Create a mock Echo context without Authorization header
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				return &c
			},
			wantErr: true,
			errMsg:  "token contains an invalid number of segments",
		},
		{
			name: "invalid token format",
			setup: func() *echo.Context {
				// Create a mock Echo context with invalid token
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer invalid-token")
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				return &c
			},
			wantErr: true,
			errMsg:  "token contains an invalid number of segments",
		},
		{
			name: "missing ca.crt in payload",
			setup: func() *echo.Context {
				// Create a mock Echo context without ca.crt in payload
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6ImtpZDEifQ.eyJpc3MiOiJ0ZXN0LWlzc3VlciIsInN1YiI6InRlc3Qtc3ViamVjdCJ9.dGhpc19pc19hX3ZhbGlkX2Jhc2U2NF9lbmNvZGVkX3NpZ25hdHVyZQ")
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)
				return &c
			},
			wantErr: true,
			errMsg:  "no jwks found",
		},
		{
			name: "successful token processing",
			setup: func() *echo.Context {
				// Generate a valid JWT token for testing
				tokenString, privateKey, err := generateTestJWT("test-issuer", "test-subject")
				if err != nil {
					s.T().Fatalf("Failed to generate test JWT: %v", err)
				}

				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer "+tokenString)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer",
					},
				}

				// Add the spec to the store
				store.Add("test-issuer", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return JWKS with our public key
				provider := &mockFederationProvider{
					jwks: map[string]map[string]string{
						"kid1": {
							"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
							"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
							"alg": "RS256",
						},
					},
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)

				// Initialize the specMap for this test
				s.server.specMap["test-ca-cert"] = []*fedv1alpha1.AuthorizationSpec{spec}

				return &c
			},
			wantIssuer: "test-issuer",
			wantSub:    "test-subject",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup the test
			c := tt.setup()

			// Call the function being tested
			issuer, sub, err := s.server.processRequest(*c)

			// Check results
			if tt.wantErr {
				s.Require().Error(err)
				if tt.errMsg != "" {
					s.Contains(err.Error(), tt.errMsg)
				}
			} else {
				s.Require().NoError(err)
				s.Equal(tt.wantIssuer, issuer)
				s.Equal(tt.wantSub, sub)
			}
		})
	}
}

func TestProcessRequestTestSuite(t *testing.T) {
	suite.Run(t, new(ProcessRequestTestSuite))
}

type GenerateSecretsTestSuite struct {
	suite.Suite
	server *ServerHandler
	specs  []*fedv1alpha1.AuthorizationSpec
}

func (s *GenerateSecretsTestSuite) SetupTest() {
	// Initialize the server handler
	s.server = NewServerHandler(nil, ":8080")

	// Initialize specs slice for cleanup
	s.specs = []*fedv1alpha1.AuthorizationSpec{}
}

func (s *GenerateSecretsTestSuite) TearDownTest() {
	// Clean up any specs added to the store
	for _, spec := range s.specs {
		store.Remove("test-issuer", spec)
	}
}

func (s *GenerateSecretsTestSuite) TestGenerateSecrets() {
	// Define test cases
	tests := []struct {
		name           string
		setup          func() echo.Context
		mockGenSecret  func(ctx context.Context, generatorName string, generatorKind string, namespace string) (map[string]string, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful secret generation",
			setup: func() echo.Context {
				// Generate a valid JWT token for testing
				tokenString, privateKey, err := generateTestJWT("test-issuer", "test-subject")
				if err != nil {
					s.T().Fatalf("Failed to generate test JWT: %v", err)
				}

				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer "+tokenString)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
				c.SetParamValues("test-generator", "test-kind", "test-namespace")

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer",
					},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{
							Name:      "test-generator",
							Kind:      "test-kind",
							Namespace: "test-namespace",
						},
					},
				}

				// Add the spec to the store
				store.Add("test-issuer", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return JWKS with our public key
				provider := &mockFederationProvider{
					jwks: map[string]map[string]string{
						"kid1": {
							"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
							"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
							"alg": "RS256",
						},
					},
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)

				// Initialize the specMap for this test
				s.server.specMap["test-ca-cert"] = []*fedv1alpha1.AuthorizationSpec{spec}

				return c
			},
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string) (map[string]string, error) {
				// Check that the parameters match what we expect
				if generatorName != "test-generator" || generatorKind != "test-kind" || namespace != "test-namespace" {
					return nil, fmt.Errorf("unexpected parameters: %s, %s, %s", generatorName, generatorKind, namespace)
				}
				// Return a mock secret
				return map[string]string{
					"key1": "value1",
					"key2": "value2",
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "{\"key1\":\"value1\",\"key2\":\"value2\"}",
		},
		{
			name: "error in processRequest",
			setup: func() echo.Context {
				// Create a mock Echo context with invalid token
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer invalid-token")
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
				c.SetParamValues("test-generator", "test-kind", "test-namespace")

				return c
			},
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string) (map[string]string, error) {
				// This should not be called
				s.T().Fatalf("mockGenSecret should not be called in this test case")
				return nil, nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "token contains an invalid number of segments",
		},
		{
			name: "no matching authorization spec",
			setup: func() echo.Context {
				// Generate a valid JWT token for testing
				tokenString, privateKey, err := generateTestJWT("test-issuer", "test-subject")
				if err != nil {
					s.T().Fatalf("Failed to generate test JWT: %v", err)
				}

				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer "+tokenString)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters with non-matching values
				c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
				c.SetParamValues("wrong-generator", "wrong-kind", "wrong-namespace")

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer",
					},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{
							Name:      "test-generator",
							Kind:      "test-kind",
							Namespace: "test-namespace",
						},
					},
				}

				// Add the spec to the store
				store.Add("test-issuer", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return JWKS with our public key
				provider := &mockFederationProvider{
					jwks: map[string]map[string]string{
						"kid1": {
							"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
							"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
							"alg": "RS256",
						},
					},
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)

				// Initialize the specMap for this test
				s.server.specMap["test-ca-cert"] = []*fedv1alpha1.AuthorizationSpec{spec}

				return c
			},
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string) (map[string]string, error) {
				// This should not be called
				s.T().Fatalf("mockGenSecret should not be called in this test case")
				return nil, nil
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Not Found",
		},
		{
			name: "error in generateSecretFn",
			setup: func() echo.Context {
				// Generate a valid JWT token for testing
				tokenString, privateKey, err := generateTestJWT("test-issuer", "test-subject")
				if err != nil {
					s.T().Fatalf("Failed to generate test JWT: %v", err)
				}

				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer "+tokenString)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
				c.SetParamValues("test-generator", "test-kind", "test-namespace")

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer",
					},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{
							Name:      "test-generator",
							Kind:      "test-kind",
							Namespace: "test-namespace",
						},
					},
				}

				// Add the spec to the store
				store.Add("test-issuer", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return JWKS with our public key
				provider := &mockFederationProvider{
					jwks: map[string]map[string]string{
						"kid1": {
							"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
							"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
							"alg": "RS256",
						},
					},
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)

				// Initialize the specMap for this test
				s.server.specMap["test-ca-cert"] = []*fedv1alpha1.AuthorizationSpec{spec}

				return c
			},
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string) (map[string]string, error) {
				return nil, fmt.Errorf("error generating secret")
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "error generating secret",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup the test
			c := tt.setup()

			// Save the original generateSecretFn
			originalGenerateSecretFn := s.server.generateSecretFn

			// Override with mock
			s.server.generateSecretFn = tt.mockGenSecret

			// Add cleanup to restore the original method after the test
			s.T().Cleanup(func() {
				s.server.generateSecretFn = originalGenerateSecretFn
			})

			// Call the function being tested
			err := s.server.generateSecrets(c)
			s.Require().NoError(err)

			// Check results
			rec := c.Response().Writer.(*httptest.ResponseRecorder)
			s.Equal(tt.expectedStatus, rec.Code)
			s.Contains(rec.Body.String(), tt.expectedBody)
		})
	}
}

func TestGenerateSecretsTestSuite(t *testing.T) {
	suite.Run(t, new(GenerateSecretsTestSuite))
}

type PostSecretsTestSuite struct {
	suite.Suite
	server *ServerHandler
	specs  []*fedv1alpha1.AuthorizationSpec
}

func (s *PostSecretsTestSuite) SetupTest() {
	// Initialize the server handler
	s.server = NewServerHandler(nil, ":8080")

	// Initialize specs slice for cleanup
	s.specs = []*fedv1alpha1.AuthorizationSpec{}
}

func (s *PostSecretsTestSuite) TearDownTest() {
	// Clean up any specs added to the store
	for _, spec := range s.specs {
		store.Remove("test-issuer", spec)
	}
}

func (s *PostSecretsTestSuite) TestPostSecrets() {
	tests := []struct {
		name           string
		setup          func() echo.Context
		mockGetSecret  func(ctx context.Context, storeName string, name string) ([]byte, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful secret retrieval",
			setup: func() echo.Context {
				// Generate a valid JWT token for testing
				tokenString, privateKey, err := generateTestJWT("test-issuer", "test-subject")
				if err != nil {
					s.T().Fatalf("Failed to generate test JWT: %v", err)
				}

				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer "+tokenString)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("secretStoreName", "secretName")
				c.SetParamValues("test-store", "test-secret")

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer",
					},
					AllowedClusterSecretStores: []string{"test-store"},
				}

				// Add the spec to the store
				store.Add("test-issuer", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return JWKS with our public key
				provider := &mockFederationProvider{
					jwks: map[string]map[string]string{
						"kid1": {
							"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
							"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
							"alg": "RS256",
						},
					},
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)

				// Initialize the specMap for this test
				s.server.specMap["test-ca-cert"] = []*fedv1alpha1.AuthorizationSpec{spec}

				return c
			},
			mockGetSecret: func(ctx context.Context, storeName string, name string) ([]byte, error) {
				// Check that the parameters match what we expect
				if storeName != "test-store" || name != "test-secret" {
					return nil, fmt.Errorf("unexpected parameters: %s, %s", storeName, name)
				}
				// Return a mock secret
				return []byte(`myvalue-is-here`), nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "myvalue-is-here",
		},
		{
			name: "error in processRequest",
			setup: func() echo.Context {
				// Create a mock Echo context with invalid token
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer invalid-token")
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("secretStoreName", "secretName")
				c.SetParamValues("test-store", "test-secret")

				return c
			},
			mockGetSecret: func(ctx context.Context, storeName string, name string) ([]byte, error) {
				// This should not be called
				s.T().Fatalf("mockGetSecret should not be called in this test case")
				return nil, nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "token contains an invalid number of segments",
		},
		{
			name: "no matching authorization spec",
			setup: func() echo.Context {
				// Generate a valid JWT token for testing
				tokenString, privateKey, err := generateTestJWT("test-issuer", "test-subject")
				if err != nil {
					s.T().Fatalf("Failed to generate test JWT: %v", err)
				}

				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer "+tokenString)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters with non-matching values
				c.SetParamNames("secretStoreName", "secretName")
				c.SetParamValues("wrong-store", "test-secret")

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer",
					},
					AllowedClusterSecretStores: []string{"test-store"},
				}

				// Add the spec to the store
				store.Add("test-issuer", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return JWKS with our public key
				provider := &mockFederationProvider{
					jwks: map[string]map[string]string{
						"kid1": {
							"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
							"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
							"alg": "RS256",
						},
					},
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)

				// Initialize the specMap for this test
				s.server.specMap["test-ca-cert"] = []*fedv1alpha1.AuthorizationSpec{spec}

				return c
			},
			mockGetSecret: func(ctx context.Context, storeName string, name string) ([]byte, error) {
				// This should not be called
				s.T().Fatalf("mockGetSecret should not be called in this test case")
				return nil, nil
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Not Found",
		},
		{
			name: "error in getSecretFn",
			setup: func() echo.Context {
				// Generate a valid JWT token for testing
				tokenString, privateKey, err := generateTestJWT("test-issuer", "test-subject")
				if err != nil {
					s.T().Fatalf("Failed to generate test JWT: %v", err)
				}

				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"ca.crt":"test-ca-cert"}`))
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				req.Header.Set("Authorization", "Bearer "+tokenString)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("secretStoreName", "secretName")
				c.SetParamValues("test-store", "test-secret")

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: fedv1alpha1.FederationSubject{
						Subject: "test-subject",
						Issuer:  "test-issuer",
					},
					AllowedClusterSecretStores: []string{"test-store"},
				}

				// Add the spec to the store
				store.Add("test-issuer", spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				// Mock a provider that will return JWKS with our public key
				provider := &mockFederationProvider{
					jwks: map[string]map[string]string{
						"kid1": {
							"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
							"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
							"alg": "RS256",
						},
					},
				}

				// Add the provider to the store
				store.AddStore(spec.FederationRef, provider)

				// Initialize the specMap for this test
				s.server.specMap["test-ca-cert"] = []*fedv1alpha1.AuthorizationSpec{spec}

				return c
			},
			mockGetSecret: func(ctx context.Context, storeName string, name string) ([]byte, error) {
				return nil, fmt.Errorf("error getting secret")
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "error getting secret",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// Setup the test
			c := tt.setup()

			// Save the original getSecretFn
			originalGetSecretFn := s.server.getSecretFn

			// Override with mock
			s.server.getSecretFn = tt.mockGetSecret

			// Add cleanup to restore the original method after the test
			s.T().Cleanup(func() {
				s.server.getSecretFn = originalGetSecretFn
			})

			// Call the function being tested
			err := s.server.postSecrets(c)
			s.Require().NoError(err)

			// Check results
			rec := c.Response().Writer.(*httptest.ResponseRecorder)
			s.Equal(tt.expectedStatus, rec.Code)
			s.Contains(rec.Body.String(), tt.expectedBody)
		})
	}
}

func TestPostSecretsTestSuite(t *testing.T) {
	suite.Run(t, new(PostSecretsTestSuite))
}

// generateTestJWT creates a signed JWT token for testing.
func generateTestJWT(issuer, subject string) (string, *rsa.PrivateKey, error) {
	// Generate a new RSA key pair for signing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, err
	}

	// Create the claims
	claims := jwt.MapClaims{
		"iss": issuer,
		"sub": subject,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "kid1"

	// Sign the token
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		return "", nil, err
	}

	return tokenString, privateKey, nil
}
