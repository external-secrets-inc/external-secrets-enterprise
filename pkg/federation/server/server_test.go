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
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/labels"

	fedv1alpha1 "github.com/external-secrets/external-secrets/apis/federation/v1alpha1"
	store "github.com/external-secrets/external-secrets/pkg/federation/store"
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
			claim, err := s.server.processRequest(*c, []byte("test-ca-cert"))

			// Check results
			if tt.wantErr {
				s.Require().Error(err)
				if tt.errMsg != "" {
					s.Contains(err.Error(), tt.errMsg)
				}
			} else {
				s.Require().NoError(err)
				s.Equal(tt.wantIssuer, claim.Issuer)
				s.Equal(tt.wantSub, claim.Subject)
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

func (s *GenerateSecretsTestSuite) generateTestSATokenWithClaims(claims KubernetesClaims, key interface{}) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // Using HS256 for simplicity with a symmetric key
	return token.SignedString(key)
}

func (s *GenerateSecretsTestSuite) TestResourcePopulationFromClaims() {
	testKey := []byte("test-symmetric-secret-key-for-hs256") // Symmetric key for HS256
	generatorName := "my-k8s-generator"
	generatorKind := "VaultGenerator"
	generatorNamespace := "secure-ns"

	baseClaims := KubernetesClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://kubernetes.default.svc.cluster.local",
			Subject:   "system:serviceaccount:kube-system:replicator",
			Audience:  jwt.ClaimStrings{"kubernetes.io/serviceaccount"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		KubernetesIOInner: KubernetesIOInner{
			Namespace: "kube-system",
			ServiceAccount: struct {
				Name string `json:"name"`
				UID  string `json:"uid"`
			}{Name: "replicator", UID: "sa-uid-replicator-777"},
		},
	}

	tests := []struct {
		name              string
		claimModifier     func(claims *KubernetesClaims) // Modifies baseClaims for the test case
		expectedOwner     string
		expectPodUID      bool
		expectedPodUID    string
		expectedSAUID     string
		expectedSAName    string
		expectedIssuer    string
		expectedNamespace string
	}{
		{
			name: "with pod information in claims",
			claimModifier: func(claims *KubernetesClaims) {
				claims.KubernetesIOInner.Pod = &struct {
					Name string `json:"name"`
					UID  string `json:"uid"`
				}{Name: "replicator-pod-xyz123", UID: "pod-uid-replicator-abc987"}
			},
			expectedOwner:     "replicator-pod-xyz123",
			expectPodUID:      true,
			expectedPodUID:    "pod-uid-replicator-abc987",
			expectedSAUID:     "sa-uid-replicator-777",
			expectedSAName:    "replicator",
			expectedIssuer:    "https://kubernetes.default.svc.cluster.local",
			expectedNamespace: "kube-system",
		},
		{
			name: "without pod information in claims",
			claimModifier: func(claims *KubernetesClaims) {
				// No pod info, baseClaims is already like this
			},
			expectedOwner:     "replicator", // Falls back to SA name
			expectPodUID:      false,
			expectedSAUID:     "sa-uid-replicator-777",
			expectedSAName:    "replicator",
			expectedIssuer:    "https://kubernetes.default.svc.cluster.local",
			expectedNamespace: "kube-system",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			currentClaims := baseClaims
			if tt.claimModifier != nil {
				tt.claimModifier(&currentClaims)
			}

			tokenString, err := s.generateTestSATokenWithClaims(currentClaims, testKey)
			s.Require().NoError(err, "Failed to generate test JWT with SA claims")

			// Mock genParseTokenFn on s.server to return a Keyfunc that "validates" our token
			originalGenParseTokenFn := s.server.genParseTokenFn
			s.server.genParseTokenFn = func(ctx context.Context, ts string, caCert []byte) func(token *jwt.Token) (interface{}, error) {
				return func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("unexpected signing method for SA token: %v", token.Header["alg"])
					}
					return testKey, nil
				}
			}
			s.T().Cleanup(func() { s.server.genParseTokenFn = originalGenParseTokenFn })

			// Create and provision AuthorizationSpec for this test case
			authSpec := &fedv1alpha1.AuthorizationSpec{
				FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-k8s-claims", Kind: "Kubernetes"},
				Subject:       fedv1alpha1.FederationSubject{Subject: currentClaims.Subject, Issuer: currentClaims.Issuer},
				AllowedGenerators: []fedv1alpha1.AllowedGenerator{
					{Name: generatorName, Kind: generatorKind, Namespace: generatorNamespace},
				},
			}
			// Use the Issuer and Subject from the spec for Set, as seen in SetupTest
			store.Add(authSpec.Subject.Issuer, authSpec)
			s.T().Cleanup(func() {
				// Remove using the issuer and spec object, as seen in user's preferred TearDownTest format
				store.Remove(authSpec.Subject.Issuer, authSpec)
			})

			var capturedResource *Resource // Variable to capture the resource

			// Mock generateSecretFn on s.server to capture the Resource and perform assertions
			originalGenerateSecretFn := s.server.generateSecretFn
			s.server.generateSecretFn = func(ctx context.Context, genName, genKind, genNamespace string, resource *Resource) (map[string]string, error) {
				s.Require().NotNil(resource, "Resource passed to generateSecretFn was nil")
				capturedResource = resource // Capture the resource
				s.Equal(generatorName, genName)
				s.Equal(generatorKind, genKind)
				s.Equal(generatorNamespace, genNamespace)
				return map[string]string{"secretKey": "secretValue"}, nil
			}
			s.T().Cleanup(func() { s.server.generateSecretFn = originalGenerateSecretFn })

			// Prepare request and context
			e := echo.New()
			// processRequest will bind the body to map[string]string for "ca.crt"
			reqBody := `{"ca.crt":"test-ca-data-for-sa-token-test"}`
			req := httptest.NewRequest(http.MethodPost, "/should_not_matter_for_handler_target", strings.NewReader(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			req.Header.Set("Authorization", "Bearer "+tokenString)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
			c.SetParamValues(generatorName, generatorKind, generatorNamespace)

			// Call the handler s.server.generateSecrets
			err = s.server.generateSecrets(c)
			s.Require().NoError(err, "s.server.generateSecrets handler returned an unexpected error")
			s.Require().Equal(http.StatusOK, rec.Code, "Expected HTTP OK status from generateSecrets")

			// Assertions on the capturedResource
			s.Require().NotNil(capturedResource, "generateSecretFn was not called or resource was not captured")
			s.Equal(generatorName, capturedResource.Name)
			s.Equal("KubernetesServiceAccount", capturedResource.AuthMethod)
			s.Equal(tt.expectedOwner, capturedResource.Owner)

			s.Equal(tt.expectedNamespace, capturedResource.OwnerAttributes["namespace"])
			s.Equal(tt.expectedIssuer, capturedResource.OwnerAttributes["issuer"])
			s.Equal(tt.expectedSAUID, capturedResource.OwnerAttributes["serviceaccount-uid"])
			s.Equal(tt.expectedSAName, capturedResource.OwnerAttributes["service-account-name"])

			if tt.expectPodUID {
				s.Equal(tt.expectedPodUID, capturedResource.OwnerAttributes["pod-uid"])
			} else {
				_, ok := capturedResource.OwnerAttributes["pod-uid"]
				s.False(ok, "pod-uid should not be present in OwnerAttributes when not in claims")
			}
		})
	}
}

func (s *GenerateSecretsTestSuite) TestRevokeSelf() {
	const (
		testIssuer        = "https://kubernetes.default.svc.cluster.local"
		testSubject       = "system:serviceaccount:test-ns:test-sa-revoke"
		testGeneratorNS   = "target-generator-ns-revoke"
		testGeneratorName = "my-revoke-generator"
		testGeneratorKind = "VaultGeneratorRevoke"
		testPodName       = "test-pod-revoke-123"
		testSAName        = "test-sa-revoke"
		testCaCertData    = "test-ca-cert-data-for-revoke-self-happy-path"
	)
	var testTokenSigningKey = []byte("test-revoke-self-secret-key-happy")

	tc := struct {
		name                  string
		setupAuthSpecs        func()
		jwtClaims             *KubernetesClaims // Using your existing KubernetesClaims struct
		expectedStatus        int
		expectDeleteCall      bool
		deleteParamsValidator func(ns string, lbls labels.Selector)
	}{
		name: "successful revocation with pod info",
		setupAuthSpecs: func() {
			authSpec := &fedv1alpha1.AuthorizationSpec{
				FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-revoke-happy", Kind: "Kubernetes"},
				Subject:       fedv1alpha1.FederationSubject{Subject: testSubject, Issuer: testIssuer},
				AllowedGenerators: []fedv1alpha1.AllowedGenerator{
					{Name: testGeneratorName, Kind: testGeneratorKind, Namespace: testGeneratorNS},
				},
			}
			store.Add(testIssuer, authSpec)
			s.T().Cleanup(func() { store.Remove(testIssuer, authSpec) })
		},
		jwtClaims: &KubernetesClaims{
			RegisteredClaims: jwt.RegisteredClaims{Issuer: testIssuer, Subject: testSubject},
			KubernetesIOInner: KubernetesIOInner{
				Namespace: "test-ns", // SA's namespace
				ServiceAccount: struct {
					Name string `json:"name"`
					UID  string `json:"uid"`
				}{Name: testSAName},
				Pod: &struct {
					Name string `json:"name"`
					UID  string `json:"uid"`
				}{Name: testPodName},
			},
		},
		expectedStatus:   http.StatusOK,
		expectDeleteCall: true,
		deleteParamsValidator: func(ns string, lbls labels.Selector) {
			s.Equal(testGeneratorNS, ns)
			expectedOwnerLabels := labels.Set{
				"federation.externalsecrets.com/owner":          testPodName,
				"federation.externalsecrets.com/generator":      testGeneratorName,
				"federation.externalsecrets.com/generator-kind": testGeneratorKind,
			}
			s.Equal(labels.SelectorFromSet(expectedOwnerLabels).String(), lbls.String())
		},
	}

	s.Run(tc.name, func() {
		// Setup: AuthSpecs in store
		tc.setupAuthSpecs()

		// Mock genParseTokenFn to handle HS256 for this test
		originalGenParseTokenFn := s.server.genParseTokenFn
		s.server.genParseTokenFn = func(ctx context.Context, onlyTokenValue string, caCrtValue []byte) func(token *jwt.Token) (interface{}, error) {
			return func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
					// testTokenSigningKey is defined at the top of TestRevokeSelf (table-driven test)
					return testTokenSigningKey, nil
				}
				// Fallback to original if not HMAC, or error, depending on test needs.
				if originalGenParseTokenFn != nil {
					return originalGenParseTokenFn(ctx, onlyTokenValue, caCrtValue)(token)
				}
				return nil, fmt.Errorf("test: unexpected signing method: %v and no original parser available", token.Header["alg"])
			}
		}
		s.T().Cleanup(func() { s.server.genParseTokenFn = originalGenParseTokenFn })

		// Generate HS256 token that processRequest can validate
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, tc.jwtClaims)
		tokenString, err := token.SignedString(testTokenSigningKey)
		s.Require().NoError(err, "Failed to sign test token")

		// Setup: Mock deleteGeneratorStateFn (ONLY this is mocked for revokeSelf internals)
		var deleteCalled bool
		var capturedDeleteNamespace string
		var capturedDeleteLabels labels.Selector
		originalDeleteFn := s.server.deleteGeneratorStateFn
		s.server.deleteGeneratorStateFn = func(ctx context.Context, namespace string, lbls labels.Selector) error {
			deleteCalled = true
			capturedDeleteNamespace = namespace
			capturedDeleteLabels = lbls
			return nil // Success for happy path
		}
		s.T().Cleanup(func() { s.server.deleteGeneratorStateFn = originalDeleteFn })

		// Prepare Echo context
		e := echo.New()
		// ca.crt in body is needed for processRequest to extract it for its jwtKeyFunc logic, even if hs256Key is set.
		reqBody := fmt.Sprintf(`{"ca.crt":%q}`, testCaCertData)
		req := httptest.NewRequest(http.MethodDelete, "/test/revoke", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("generatorNamespace", "generatorName", "generatorKind")
		c.SetParamValues(testGeneratorNS, testGeneratorName, testGeneratorKind)

		// Call the handler (revokeSelf)
		handlerErr := s.server.revokeSelf(c) // processRequest is called internally and is NOT mocked
		s.Require().NoError(handlerErr, "Handler invocation itself should not error out")

		// Assertions
		s.Equal(tc.expectedStatus, rec.Code)
		s.Equal(tc.expectDeleteCall, deleteCalled, "deleteGeneratorStateFn call expectation mismatch")

		if tc.expectDeleteCall && tc.deleteParamsValidator != nil {
			tc.deleteParamsValidator(capturedDeleteNamespace, capturedDeleteLabels)
		}
	})
}

func (s *GenerateSecretsTestSuite) TestRevokeSelfHappyPath() {
	const (
		testIssuer        = "https://kubernetes.default.svc.cluster.local/revoke-self-happy"
		testSubject       = "system:serviceaccount:test-ns:test-sa-revoke-happy"
		testGeneratorNS   = "target-generator-ns-revoke-happy"
		testGeneratorName = "my-revoke-generator-happy"
		testGeneratorKind = "VaultGeneratorRevokeHappy"
		testPodName       = "test-pod-revoke-happy-123"
		testSAName        = "test-sa-revoke-happy"
		testCaCertData    = "test-ca-cert-data-for-revoke-self-happy-path"
	)
	var testTokenSigningKey = []byte("test-revoke-self-secret-key-hs256-happy")

	s.Run("successful revocation with pod info", func() {
		// 1. Setup AuthorizationSpec in store
		authSpec := &fedv1alpha1.AuthorizationSpec{
			FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-revoke-happy-path", Kind: "Kubernetes"},
			Subject:       fedv1alpha1.FederationSubject{Subject: testSubject, Issuer: testIssuer},
			AllowedGenerators: []fedv1alpha1.AllowedGenerator{
				{Name: testGeneratorName, Kind: testGeneratorKind, Namespace: testGeneratorNS},
			},
		}
		store.Add(testIssuer, authSpec)
		s.T().Cleanup(func() { store.Remove(testIssuer, authSpec) })

		// 2. Mock genParseTokenFn to handle HS256 for this test
		originalGenParseTokenFn := s.server.genParseTokenFn
		s.server.genParseTokenFn = func(ctx context.Context, onlyTokenValue string, caCrtValue []byte) func(token *jwt.Token) (interface{}, error) {
			return func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
					// testTokenSigningKey is defined at the top of TestRevokeSelfHappyPath
					return testTokenSigningKey, nil
				}
				// Fallback for safety, though this test specifically uses HS256
				if originalGenParseTokenFn != nil {
					return originalGenParseTokenFn(ctx, onlyTokenValue, caCrtValue)(token)
				}
				return nil, fmt.Errorf("test: unexpected signing method: %v and no original parser available", token.Header["alg"])
			}
		}
		s.T().Cleanup(func() { s.server.genParseTokenFn = originalGenParseTokenFn })

		// 3. Prepare JWT Claims and generate HS256 token
		claims := &KubernetesClaims{ // Assuming KubernetesClaims struct is defined in this file/package
			RegisteredClaims: jwt.RegisteredClaims{Issuer: testIssuer, Subject: testSubject},
			KubernetesIOInner: KubernetesIOInner{
				Namespace: "test-ns", // SA's namespace
				ServiceAccount: struct {
					Name string `json:"name"`
					UID  string `json:"uid"`
				}{Name: testSAName, UID: "sa-uid-revoke-happy"},
				Pod: &struct {
					Name string `json:"name"`
					UID  string `json:"uid"`
				}{Name: testPodName, UID: "pod-uid-revoke-happy"},
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(testTokenSigningKey)
		s.Require().NoError(err, "Failed to sign test token for revokeSelf")

		// 4. Mock deleteGeneratorStateFn (ONLY this is mocked for revokeSelf internals)
		var deleteCalled bool
		var capturedDeleteNamespace string
		var capturedDeleteLabels labels.Selector
		originalDeleteFn := s.server.deleteGeneratorStateFn
		s.server.deleteGeneratorStateFn = func(ctx context.Context, namespace string, lbls labels.Selector) error {
			deleteCalled = true
			capturedDeleteNamespace = namespace
			capturedDeleteLabels = lbls
			return nil // Success for happy path
		}
		s.T().Cleanup(func() { s.server.deleteGeneratorStateFn = originalDeleteFn })

		// 5. Prepare Echo context
		e := echo.New()
		// ca.crt in body is needed for processRequest to extract it for its jwtKeyFunc logic (even if hs256Key is set on server).
		reqBody := fmt.Sprintf(`{"ca.crt":%q}`, testCaCertData)
		req := httptest.NewRequest(http.MethodDelete, "/test/revokeSelfHappyPath", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("generatorNamespace", "generatorName", "generatorKind")
		c.SetParamValues(testGeneratorNS, testGeneratorName, testGeneratorKind)

		// 6. Call the handler (revokeSelf)
		// processRequest is called internally by revokeSelf and is NOT mocked here.
		handlerErr := s.server.revokeSelf(c)
		s.Require().NoError(handlerErr, "Handler invocation itself should not error out in happy path")

		// 7. Assertions
		s.Equal(http.StatusOK, rec.Code, "Expected HTTP OK status")
		s.True(deleteCalled, "deleteGeneratorStateFn should have been called")

		if deleteCalled { // Only validate params if called, to avoid nil pointer if test setup fails earlier
			s.Equal(testGeneratorNS, capturedDeleteNamespace, "Incorrect namespace passed to deleteGeneratorStateFn")
			expectedOwnerLabels := labels.Set{
				"federation.externalsecrets.com/owner":          testPodName,
				"federation.externalsecrets.com/generator":      testGeneratorName,
				"federation.externalsecrets.com/generator-kind": testGeneratorKind,
			}
			s.Equal(labels.SelectorFromSet(expectedOwnerLabels).String(), capturedDeleteLabels.String(), "Incorrect labels passed to deleteGeneratorStateFn")
		}
	})
}

func (s *GenerateSecretsTestSuite) TestGenerateSecrets() {
	tests := []struct {
		name           string
		setup          func() echo.Context
		mockGenSecret  func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, error)
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
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, error) {
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
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, error) {
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
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, error) {
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
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, error) {
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

func (s *GenerateSecretsTestSuite) TestRevokeCredentialsOfHappyPath() {
	const (
		testIssuer           = "https://kubernetes.default.svc.cluster.local/revoke-creds-happy"
		testSubject          = "system:serviceaccount:test-ns:test-sa-revoke-creds-happy"
		testParamGeneratorNS = "param-generator-ns-revoke-creds-happy" // Namespace from path param
		testReqOwner         = "test-pod-revoke-creds-happy-456"       // Owner from request body
		testReqDeleteNS      = "target-delete-ns-revoke-creds-happy"   // Namespace for deletion from request body
		testCaCertData       = "test-ca-cert-data-for-revoke-creds-happy"
		testSAName           = "test-sa-revoke-creds-happy"
	)
	var testTokenSigningKey = []byte("test-revoke-creds-secret-key-hs256-happy")

	s.Run("successful revocation of credentials", func() {
		// 1. Setup AuthorizationSpec in store
		authSpec := &fedv1alpha1.AuthorizationSpec{
			FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-revoke-creds-happy", Kind: "Kubernetes"},
			Subject:       fedv1alpha1.FederationSubject{Subject: testSubject, Issuer: testIssuer},
			AllowedGeneratorStates: []fedv1alpha1.AllowedGeneratorState{ // Used by revokeCredentialsOf
				{Namespace: testParamGeneratorNS},
			},
		}
		store.Add(testIssuer, authSpec)
		s.T().Cleanup(func() { store.Remove(testIssuer, authSpec) })

		// 2. Mock genParseTokenFn to handle HS256 for this test
		originalGenParseTokenFn := s.server.genParseTokenFn
		s.server.genParseTokenFn = func(ctx context.Context, onlyTokenValue string, caCrtValue []byte) func(token *jwt.Token) (interface{}, error) {
			return func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
					return testTokenSigningKey, nil
				}
				return nil, fmt.Errorf("test: unexpected signing method: %v for revokeCredentialsOf", token.Header["alg"])
			}
		}
		s.T().Cleanup(func() { s.server.genParseTokenFn = originalGenParseTokenFn })

		// 3. Prepare JWT Claims and generate HS256 token
		claims := &KubernetesClaims{
			RegisteredClaims: jwt.RegisteredClaims{Issuer: testIssuer, Subject: testSubject},
			KubernetesIOInner: KubernetesIOInner{
				Namespace: "test-ns", // SA's namespace
				ServiceAccount: struct {
					Name string `json:"name"`
					UID  string `json:"uid"`
				}{Name: testSAName, UID: "sa-uid-revoke-creds-happy"},
				// Pod info not strictly necessary for revokeCredentialsOf logic itself, but good for consistency
				Pod: &struct {
					Name string `json:"name"`
					UID  string `json:"uid"`
				}{Name: testReqOwner, UID: "pod-uid-revoke-creds-happy"},
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(testTokenSigningKey)
		s.Require().NoError(err, "Failed to sign test token for revokeCredentialsOf")

		// 4. Mock deleteGeneratorStateFn
		var deleteCalled bool
		var capturedDeleteNamespace string
		var capturedDeleteLabels labels.Selector
		originalDeleteFn := s.server.deleteGeneratorStateFn
		s.server.deleteGeneratorStateFn = func(ctx context.Context, namespace string, lbls labels.Selector) error {
			deleteCalled = true
			capturedDeleteNamespace = namespace
			capturedDeleteLabels = lbls
			return nil // Success for happy path
		}
		s.T().Cleanup(func() { s.server.deleteGeneratorStateFn = originalDeleteFn })

		// 5. Prepare Echo context
		e := echo.New()
		// Body for revokeCredentialsOf includes owner, namespace (for deletion), and ca.crt (for HS256 key)
		reqBody := fmt.Sprintf(`{"owner":%q, "namespace":%q, "ca.crt":%q}`,
			testReqOwner, testReqDeleteNS, testCaCertData)
		req := httptest.NewRequest(http.MethodDelete, "/test/revokeCredentialsOfHappyPath", strings.NewReader(reqBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set("Authorization", "Bearer "+tokenString)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("generatorNamespace") // revokeCredentialsOf uses this path param
		c.SetParamValues(testParamGeneratorNS)

		// 6. Call the handler
		handlerErr := s.server.revokeCredentialsOf(c)
		s.Require().NoError(handlerErr, "Handler invocation should not error in happy path for revokeCredentialsOf")

		// 7. Assertions
		s.Equal(http.StatusOK, rec.Code, "Expected HTTP OK status for revokeCredentialsOf")
		s.True(deleteCalled, "deleteGeneratorStateFn should have been called for revokeCredentialsOf")

		if deleteCalled {
			s.Equal(testReqDeleteNS, capturedDeleteNamespace, "Incorrect namespace passed to deleteGeneratorStateFn")
			expectedOwnerLabels := labels.Set{
				"federation.externalsecrets.com/owner": testReqOwner,
			}
			s.Equal(labels.SelectorFromSet(expectedOwnerLabels).String(), capturedDeleteLabels.String(), "Incorrect labels passed to deleteGeneratorStateFn")
		}
	})
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
