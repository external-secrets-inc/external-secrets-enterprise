// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/suite"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fedv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/federation/v1alpha1"
	externalsecrets "github.com/external-secrets/external-secrets/pkg/controllers/externalsecret"
	"github.com/external-secrets/external-secrets/pkg/enterprise/federation/server/auth"
	store "github.com/external-secrets/external-secrets/pkg/enterprise/federation/store"
)

type GenerateSecretsTestSuite struct {
	suite.Suite
	server *ServerHandler
	specs  []*fedv1alpha1.AuthorizationSpec
}

func (s *GenerateSecretsTestSuite) SetupTest() {
	// Initialize the server handler
	s.server = NewServerHandler(nil, ":8080", ":8081", "unix:///spire.sock", true)

	// Initialize specs slice for cleanup
	s.specs = []*fedv1alpha1.AuthorizationSpec{}
}

func (s *GenerateSecretsTestSuite) TearDownTest() {
	// Clean up any specs added to the store
	for _, spec := range s.specs {
		store.Remove("test-issuer", spec)
	}
}

func (s *GenerateSecretsTestSuite) TestResourcePopulationFromClaims() {
	generatorName := "my-k8s-generator"
	generatorKind := "VaultGenerator"
	generatorNamespace := "secure-ns"

	tests := []struct {
		name              string
		authInfo          *auth.AuthInfo
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
			authInfo: &auth.AuthInfo{
				Method:   "oidc",
				Provider: "https://kubernetes.default.svc.cluster.local",
				Subject:  "system:serviceaccount:kube-system:replicator",
				KubeAttributes: &auth.KubeAttributes{
					Namespace: "kube-system",
					ServiceAccount: &auth.ServiceAccount{
						Name: "replicator",
						UID:  "sa-uid-replicator-777",
					},
					Pod: &auth.PodInfo{
						Name: "replicator-pod-xyz123",
						UID:  "pod-uid-replicator-abc987",
					},
				},
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
			authInfo: &auth.AuthInfo{
				Method:   "oidc",
				Provider: "https://kubernetes.default.svc.cluster.local",
				Subject:  "system:serviceaccount:kube-system:replicator",
				KubeAttributes: &auth.KubeAttributes{
					Namespace: "kube-system",
					ServiceAccount: &auth.ServiceAccount{
						Name: "replicator",
						UID:  "sa-uid-replicator-777",
					},
				},
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
			// Create and provision AuthorizationSpec for this test case
			authSpec := &fedv1alpha1.AuthorizationSpec{
				FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-k8s-claims", Kind: "Kubernetes"},
				Subject:       &fedv1alpha1.FederationSubject{Subject: tt.authInfo.Subject, Issuer: tt.authInfo.Provider},
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
			s.server.generateSecretFn = func(ctx context.Context, genName, genKind, genNamespace string, resource *Resource) (map[string]string, string, string, error) {
				s.Require().NotNil(resource, "Resource passed to generateSecretFn was nil")
				capturedResource = resource // Capture the resource
				s.Equal(generatorName, genName)
				s.Equal(generatorKind, genKind)
				s.Equal(generatorNamespace, genNamespace)
				return map[string]string{"secretKey": "secretValue"}, "test-state", "test-namespace", nil
			}
			s.T().Cleanup(func() { s.server.generateSecretFn = originalGenerateSecretFn })

			// Prepare request and context
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/should_not_matter_for_handler_target", http.NoBody)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
			c.SetParamValues(generatorName, generatorKind, generatorNamespace)
			c.Set("authInfo", tt.authInfo)

			// Call the handler s.server.generateSecrets
			err := s.server.generateSecrets(c)
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

	tc := []struct {
		name                  string
		setupAuthSpecs        func()
		authInfo              *auth.AuthInfo
		expectedStatus        int
		expectDeleteCall      bool
		deleteParamsValidator func(ns string, lbls labels.Selector)
	}{
		{
			name: "successful revocation with pod info",
			setupAuthSpecs: func() {
				authSpec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-revoke-happy", Kind: "Kubernetes"},
					Subject:       &fedv1alpha1.FederationSubject{Subject: testSubject, Issuer: testIssuer},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{Name: testGeneratorName, Kind: testGeneratorKind, Namespace: testGeneratorNS},
					},
				}
				store.Add(testIssuer, authSpec)
				s.T().Cleanup(func() { store.Remove(testIssuer, authSpec) })
			},
			authInfo: &auth.AuthInfo{
				Method:   "oidc",
				Provider: testIssuer,
				Subject:  testSubject,
				KubeAttributes: &auth.KubeAttributes{
					Namespace: "test-ns",
					ServiceAccount: &auth.ServiceAccount{
						Name: testSAName,
					},
					Pod: &auth.PodInfo{
						Name: testPodName,
					},
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
		}, {
			name: "failed revocation without kubernetes attributes",
			setupAuthSpecs: func() {
				authSpec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-revoke-happy", Kind: "Kubernetes"},
					Subject:       &fedv1alpha1.FederationSubject{Subject: testSubject, Issuer: testIssuer},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{Name: testGeneratorName, Kind: testGeneratorKind, Namespace: testGeneratorNS},
					},
				}
				store.Add(testIssuer, authSpec)
				s.T().Cleanup(func() { store.Remove(testIssuer, authSpec) })
			},
			authInfo: &auth.AuthInfo{
				Method:   "oidc",
				Provider: testIssuer,
				Subject:  testSubject,
			},
			expectedStatus:   http.StatusBadRequest,
			expectDeleteCall: false,
			deleteParamsValidator: func(ns string, lbls labels.Selector) {
				s.Equal(testGeneratorNS, ns)
				expectedOwnerLabels := labels.Set{
					"federation.externalsecrets.com/owner":          testPodName,
					"federation.externalsecrets.com/generator":      testGeneratorName,
					"federation.externalsecrets.com/generator-kind": testGeneratorKind,
				}
				s.Equal(labels.SelectorFromSet(expectedOwnerLabels).String(), lbls.String())
			},
		}, {
			name: "failed revocation without kubernetes service account",
			setupAuthSpecs: func() {
				authSpec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-revoke-happy", Kind: "Kubernetes"},
					Subject:       &fedv1alpha1.FederationSubject{Subject: testSubject, Issuer: testIssuer},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{Name: testGeneratorName, Kind: testGeneratorKind, Namespace: testGeneratorNS},
					},
				}
				store.Add(testIssuer, authSpec)
				s.T().Cleanup(func() { store.Remove(testIssuer, authSpec) })
			},
			authInfo: &auth.AuthInfo{
				Method:   "oidc",
				Provider: testIssuer,
				Subject:  testSubject,
				KubeAttributes: &auth.KubeAttributes{
					Namespace: "test-ns",
				},
			},
			expectedStatus:   http.StatusBadRequest,
			expectDeleteCall: false,
			deleteParamsValidator: func(ns string, lbls labels.Selector) {
				s.Equal(testGeneratorNS, ns)
				expectedOwnerLabels := labels.Set{
					"federation.externalsecrets.com/owner":          testPodName,
					"federation.externalsecrets.com/generator":      testGeneratorName,
					"federation.externalsecrets.com/generator-kind": testGeneratorKind,
				}
				s.Equal(labels.SelectorFromSet(expectedOwnerLabels).String(), lbls.String())
			},
		},
	}

	for _, tt := range tc {
		s.Run(tt.name, func() {
			// Setup: AuthSpecs in store
			tt.setupAuthSpecs()

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
			req := httptest.NewRequest(http.MethodDelete, "/test/revoke", http.NoBody)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("generatorNamespace", "generatorName", "generatorKind")
			c.SetParamValues(testGeneratorNS, testGeneratorName, testGeneratorKind)
			c.Set("authInfo", tt.authInfo)

			// Call the handler (revokeSelf)
			handlerErr := s.server.revokeSelf(c) // processRequest is called internally and is NOT mocked
			s.Require().NoError(handlerErr, "Handler invocation itself should not error out")

			// Assertions
			s.Equal(tt.expectedStatus, rec.Code)
			s.Equal(tt.expectDeleteCall, deleteCalled, "deleteGeneratorStateFn call expectation mismatch")

			if tt.expectDeleteCall && tt.deleteParamsValidator != nil {
				tt.deleteParamsValidator(capturedDeleteNamespace, capturedDeleteLabels)
			}
		})
	}
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
	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: testIssuer,
		Subject:  testSubject,
		KubeAttributes: &auth.KubeAttributes{
			Namespace: "test-ns",
			ServiceAccount: &auth.ServiceAccount{
				Name: testSAName,
				UID:  "sa-uid-revoke-happy",
			},
			Pod: &auth.PodInfo{
				Name: testPodName,
				UID:  "pod-uid-revoke-happy",
			},
		},
	}

	s.Run("successful revocation with pod info", func() {
		// 1. Setup AuthorizationSpec in store
		authSpec := &fedv1alpha1.AuthorizationSpec{
			FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-revoke-happy-path", Kind: "Kubernetes"},
			Subject:       &fedv1alpha1.FederationSubject{Subject: testSubject, Issuer: testIssuer},
			AllowedGenerators: []fedv1alpha1.AllowedGenerator{
				{Name: testGeneratorName, Kind: testGeneratorKind, Namespace: testGeneratorNS},
			},
		}
		store.Add(testIssuer, authSpec)
		s.T().Cleanup(func() { store.Remove(testIssuer, authSpec) })

		// 2. Mock deleteGeneratorStateFn (ONLY this is mocked for revokeSelf internals)
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

		// 3. Prepare Echo context
		e := echo.New()
		req := httptest.NewRequest(http.MethodDelete, "/test/revokeSelfHappyPath", http.NoBody)
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("generatorNamespace", "generatorName", "generatorKind")
		c.SetParamValues(testGeneratorNS, testGeneratorName, testGeneratorKind)
		c.Set("authInfo", authInfo)

		// 4. Call the handler (revokeSelf)
		// processRequest is called internally by revokeSelf and is NOT mocked here.
		handlerErr := s.server.revokeSelf(c)
		s.Require().NoError(handlerErr, "Handler invocation itself should not error out in happy path")

		// 5. Assertions
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
	const (
		testIssuer    = "test-issuer"
		testSubject   = "test-subject"
		testNamespace = "test-namespace"
	)

	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: testIssuer,
		Subject:  testSubject,
		KubeAttributes: &auth.KubeAttributes{
			Namespace: testNamespace,
			ServiceAccount: &auth.ServiceAccount{
				Name: "test-service-account",
				UID:  "test-service-account-uid",
			},
		},
	}

	tests := []struct {
		name           string
		setup          func() echo.Context
		mockGenSecret  func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, string, string, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful secret generation",
			setup: func() echo.Context {
				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
				c.SetParamValues("test-generator", "test-kind", testNamespace)
				c.Set("authInfo", authInfo)

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: &fedv1alpha1.FederationSubject{
						Subject: testSubject,
						Issuer:  testIssuer,
					},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{
							Name:      "test-generator",
							Kind:      "test-kind",
							Namespace: testNamespace,
						},
					},
				}

				// Add the spec to the store
				store.Add(testIssuer, spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				return c
			},
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, string, string, error) {
				// Check that the parameters match what we expect
				if generatorName != "test-generator" || generatorKind != "test-kind" || namespace != testNamespace {
					return nil, "", "", fmt.Errorf("unexpected parameters: %s, %s, %s", generatorName, generatorKind, namespace)
				}

				return map[string]string{
					"key1": "value1",
					"key2": "value2",
				}, "test-state", testNamespace, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "{\"key1\":\"value1\",\"key2\":\"value2\"}",
		},
		{
			name: "no matching authorization spec",
			setup: func() echo.Context {
				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters with non-matching values
				c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
				c.SetParamValues("wrong-generator", "wrong-kind", "wrong-namespace")
				c.Set("authInfo", authInfo)

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: &fedv1alpha1.FederationSubject{
						Subject: testSubject,
						Issuer:  testIssuer,
					},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{
							Name:      "test-generator",
							Kind:      "test-kind",
							Namespace: testNamespace,
						},
					},
				}

				// Add the spec to the store
				store.Add(testIssuer, spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)
				return c
			},
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, string, string, error) {
				// This should not be called
				s.T().Fatalf("mockGenSecret should not be called in this test case")
				return nil, "", "", nil
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "Not Found",
		},
		{
			name: "error in generateSecretFn",
			setup: func() echo.Context {
				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
				c.SetParamValues("test-generator", "test-kind", testNamespace)
				c.Set("authInfo", authInfo)

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: &fedv1alpha1.FederationSubject{
						Subject: testSubject,
						Issuer:  testIssuer,
					},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{
							Name:      "test-generator",
							Kind:      "test-kind",
							Namespace: testNamespace,
						},
					},
				}

				// Add the spec to the store
				store.Add(testIssuer, spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				return c
			},
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, string, string, error) {
				return nil, "", "", fmt.Errorf("error generating secret")
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "error generating secret",
		},
		{
			name: "error missing kubernetes attributes",
			setup: func() echo.Context {
				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
				c.SetParamValues("test-generator", "test-kind", testNamespace)

				customAuthInfo := &auth.AuthInfo{
					Method:   "oidc",
					Provider: testIssuer,
					Subject:  testSubject,
				}
				c.Set("authInfo", customAuthInfo)

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: &fedv1alpha1.FederationSubject{
						Subject: testSubject,
						Issuer:  testIssuer,
					},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{
							Name:      "test-generator",
							Kind:      "test-kind",
							Namespace: testNamespace,
						},
					},
				}

				// Add the spec to the store
				store.Add(testIssuer, spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				return c
			},
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, string, string, error) {
				// This should not be called
				s.T().Fatalf("mockGenSecret should not be called in this test case")
				return nil, "", "", nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "missing kubernetes attributes",
		},
		{
			name: "error missing kubernetes attributes",
			setup: func() echo.Context {
				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("generatorName", "generatorKind", "generatorNamespace")
				c.SetParamValues("test-generator", "test-kind", testNamespace)

				customAuthInfo := &auth.AuthInfo{
					Method:   "oidc",
					Provider: testIssuer,
					Subject:  testSubject,
					KubeAttributes: &auth.KubeAttributes{
						Namespace: testNamespace,
					},
				}
				c.Set("authInfo", customAuthInfo)

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: &fedv1alpha1.FederationSubject{
						Subject: testSubject,
						Issuer:  testIssuer,
					},
					AllowedGenerators: []fedv1alpha1.AllowedGenerator{
						{
							Name:      "test-generator",
							Kind:      "test-kind",
							Namespace: testNamespace,
						},
					},
				}

				// Add the spec to the store
				store.Add(testIssuer, spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

				return c
			},
			mockGenSecret: func(ctx context.Context, generatorName string, generatorKind string, namespace string, resource *Resource) (map[string]string, string, string, error) {
				// This should not be called
				s.T().Fatalf("mockGenSecret should not be called in this test case")
				return nil, "", "", nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "missing kubernetes service account",
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

	s.Run("successful revocation of credentials", func() {
		// 1. Setup AuthorizationSpec in store
		authSpec := &fedv1alpha1.AuthorizationSpec{
			FederationRef: fedv1alpha1.FederationRef{Name: "test-fed-revoke-creds-happy", Kind: "Kubernetes"},
			Subject:       &fedv1alpha1.FederationSubject{Subject: testSubject, Issuer: testIssuer},
			AllowedGeneratorStates: []fedv1alpha1.AllowedGeneratorState{ // Used by revokeCredentialsOf
				{Namespace: testParamGeneratorNS},
			},
		}
		store.Add(testIssuer, authSpec)
		s.T().Cleanup(func() { store.Remove(testIssuer, authSpec) })

		// 2. Setup auth info
		authInfo := &auth.AuthInfo{
			Method:   "oidc",
			Provider: testIssuer,
			Subject:  testSubject,
			KubeAttributes: &auth.KubeAttributes{
				Namespace: "test-ns",
				ServiceAccount: &auth.ServiceAccount{
					Name: testSAName,
					UID:  "sa-uid-revoke-creds-happy",
				},
				Pod: &auth.PodInfo{
					Name: testReqOwner,
					UID:  "pod-uid-revoke-creds-happy",
				},
			},
		}

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
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.SetParamNames("generatorNamespace") // revokeCredentialsOf uses this path param
		c.SetParamValues(testParamGeneratorNS)
		c.Set("authInfo", authInfo)

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
	s.server = NewServerHandler(nil, ":8080", ":8081", "unix:///spire.sock", true)

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
	const (
		testIssuer  = "test-issuer"
		testSubject = "test-subject"
	)

	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: testIssuer,
		Subject:  testSubject,
	}
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
				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("secretStoreName", "secretName")
				c.SetParamValues("test-store", "test-secret")
				c.Set("authInfo", authInfo)

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: &fedv1alpha1.FederationSubject{
						Subject: testSubject,
						Issuer:  testIssuer,
					},
					AllowedClusterSecretStores: []string{"test-store"},
				}

				// Add the spec to the store
				store.Add(testIssuer, spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

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
			name: "no matching authorization spec",
			setup: func() echo.Context {
				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters with non-matching values
				c.SetParamNames("secretStoreName", "secretName")
				c.SetParamValues("wrong-store", "test-secret")
				c.Set("authInfo", authInfo)

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: &fedv1alpha1.FederationSubject{
						Subject: testSubject,
						Issuer:  testIssuer,
					},
					AllowedClusterSecretStores: []string{"test-store"},
				}

				// Add the spec to the store
				store.Add(testIssuer, spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

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
				// Create a mock Echo context
				e := echo.New()
				req := httptest.NewRequest(http.MethodPost, "/", http.NoBody)
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
				rec := httptest.NewRecorder()
				c := e.NewContext(req, rec)

				// Set path parameters
				c.SetParamNames("secretStoreName", "secretName")
				c.SetParamValues("test-store", "test-secret")
				c.Set("authInfo", authInfo)

				// Setup the server for this test
				spec := &fedv1alpha1.AuthorizationSpec{
					FederationRef: fedv1alpha1.FederationRef{
						Name: "test-federation",
						Kind: "Kubernetes",
					},
					Subject: &fedv1alpha1.FederationSubject{
						Subject: testSubject,
						Issuer:  testIssuer,
					},
					AllowedClusterSecretStores: []string{"test-store"},
				}

				// Add the spec to the store
				store.Add(testIssuer, spec)

				// Store the spec for cleanup
				s.specs = append(s.specs, spec)

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

type fakeAuthProvider struct {
	info *auth.AuthInfo
	err  error
}

func (f *fakeAuthProvider) Authenticate(req *http.Request) (*auth.AuthInfo, error) {
	return f.info, f.err
}

type AuthMiddlewareSuite struct {
	suite.Suite
	server       *ServerHandler
	origRegistry map[string]auth.Authenticator
}

func (s *AuthMiddlewareSuite) SetupTest() {
	s.server = &ServerHandler{}

	s.origRegistry = auth.Registry
}

func (s *AuthMiddlewareSuite) TearDownTest() {
	auth.Registry = s.origRegistry
}

func (s *AuthMiddlewareSuite) Test_FirstProviderSucceeds() {
	expected := &auth.AuthInfo{Method: "oidc", Provider: "test", Subject: "xyz"}
	auth.Registry = map[string]auth.Authenticator{
		"test": &fakeAuthProvider{info: expected, err: nil},
	}

	nextCalled := false
	next := echo.HandlerFunc(func(c echo.Context) error {
		nextCalled = true
		s.Equal(expected, c.Get("authInfo"))
		return c.String(http.StatusOK, "ok")
	})

	mw := s.server.authMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	err := mw(c)
	s.NoError(err)
	s.True(nextCalled, "handler wasn't called")
	s.Equal(http.StatusOK, rec.Code)
	s.Equal("ok", rec.Body.String())
}

func (s *AuthMiddlewareSuite) Test_SecondProviderSucceeds() {
	expected := &auth.AuthInfo{Method: "oidc", Provider: "test", Subject: "xyz"}
	auth.Registry = map[string]auth.Authenticator{
		"first":  &fakeAuthProvider{info: nil, err: errors.New("err1")},
		"second": &fakeAuthProvider{info: expected, err: nil},
	}

	nextCalled := false
	next := echo.HandlerFunc(func(c echo.Context) error {
		nextCalled = true
		s.Equal(expected, c.Get("authInfo"))
		return c.JSON(http.StatusAccepted, expected)
	})

	mw := s.server.authMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	err := mw(c)
	s.NoError(err)
	s.True(nextCalled, "handler wasn't called")
	s.Equal(http.StatusAccepted, rec.Code)
}

func (s *AuthMiddlewareSuite) Test_AllProvidersFail() {
	auth.Registry = map[string]auth.Authenticator{
		"first":  &fakeAuthProvider{info: nil, err: errors.New("errA")},
		"second": &fakeAuthProvider{info: nil, err: errors.New("errB")},
	}

	next := echo.HandlerFunc(func(c echo.Context) error {
		s.Fail("should not be called when no auth provider succeeds")
		return nil
	})

	mw := s.server.authMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	err := mw(c)
	s.NoError(err)
	s.Equal(http.StatusUnauthorized, rec.Code)
	s.Equal(`"errB"`+"\n", rec.Body.String())
}

func TestAuthMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(AuthMiddlewareSuite))
}

// mockClient is a mock implementation of client.Client for testing.
type mockClient struct {
	getErr         error
	createErr      error
	updateErr      error
	getCalled      bool
	createCalled   bool
	updateCalled   bool
	storedIdentity *fedv1alpha1.AuthorizedIdentity
}

func (m *mockClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	m.getCalled = true
	if m.getErr != nil {
		return m.getErr
	}
	// If we have a stored identity, copy it to obj
	if m.storedIdentity != nil {
		if identity, ok := obj.(*fedv1alpha1.AuthorizedIdentity); ok {
			*identity = *m.storedIdentity.DeepCopy()
		}
	}
	return nil
}

func (m *mockClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return nil
}

func (m *mockClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	m.createCalled = true
	if m.createErr != nil {
		return m.createErr
	}
	// Store the created identity
	if identity, ok := obj.(*fedv1alpha1.AuthorizedIdentity); ok {
		m.storedIdentity = identity.DeepCopy()
	}
	return nil
}

func (m *mockClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	return nil
}

func (m *mockClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	m.updateCalled = true
	if m.updateErr != nil {
		return m.updateErr
	}
	// Update the stored identity
	if identity, ok := obj.(*fedv1alpha1.AuthorizedIdentity); ok {
		m.storedIdentity = identity.DeepCopy()
	}
	return nil
}

func (m *mockClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return nil
}

func (m *mockClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (m *mockClient) Status() client.StatusWriter {
	return nil
}

func (m *mockClient) Scheme() *runtime.Scheme {
	return nil
}

func (m *mockClient) RESTMapper() meta.RESTMapper {
	return nil
}

func (m *mockClient) SubResource(subResource string) client.SubResourceClient {
	return nil
}

func (m *mockClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return schema.GroupVersionKind{}, nil
}

func (m *mockClient) IsObjectNamespaced(obj runtime.Object) (bool, error) {
	return false, nil
}

func (m *mockClient) Apply(ctx context.Context, obj runtime.ApplyConfiguration, opts ...client.ApplyOption) error {
	return nil
}

func TestUpsertIdentityConnectionError(t *testing.T) {
	// Test that upsertIdentity returns early when Get returns a non-NotFound error
	ctx := context.Background()

	// Create a mock client that returns a connection error
	connectionErr := errors.New("connection refused")
	mockClient := &mockClient{getErr: connectionErr}

	// Create a minimal reconciler with the mock client
	reconciler := &externalsecrets.Reconciler{
		Client: mockClient,
	}

	// Create the server handler with the mock reconciler
	server := &ServerHandler{
		reconciler: reconciler,
	}

	// Create test auth info
	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: "test-provider",
		Subject:  "test-subject",
		KubeAttributes: &auth.KubeAttributes{
			Namespace: "test-ns",
			ServiceAccount: &auth.ServiceAccount{
				Name: "test-sa",
				UID:  "test-uid",
			},
		},
	}

	// Create test parameters
	federationRef := &fedv1alpha1.FederationRef{
		Kind: "Kubernetes",
		Name: "test-federation",
	}

	// Call upsertIdentity
	err := server.upsertIdentity(
		ctx,
		authInfo,
		federationRef,
		"test-generator",
		"test-key",
		"Generator",
		"test-namespace",
		nil,
	)

	// Assert that the error is returned (should contain the connection error)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	// Verify the error message contains our connection error
	if !strings.Contains(err.Error(), "failed to get AuthorizedIdentity") {
		t.Errorf("expected error to contain 'failed to get AuthorizedIdentity', got: %v", err)
	}
}

func TestUpsertIdentityCreateNew(t *testing.T) {
	// Test that upsertIdentity creates a new AuthorizedIdentity when it doesn't exist
	ctx := context.Background()

	// Create a mock client that returns NotFound error
	notFoundErr := apierrors.NewNotFound(schema.GroupResource{Group: "federation.external-secrets.io", Resource: "authorizedidentities"}, "test-identity")
	mockClient := &mockClient{getErr: notFoundErr}

	reconciler := &externalsecrets.Reconciler{
		Client: mockClient,
	}

	server := &ServerHandler{
		reconciler: reconciler,
	}

	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: "test-provider",
		Subject:  "test-subject",
		KubeAttributes: &auth.KubeAttributes{
			Namespace: "test-ns",
			ServiceAccount: &auth.ServiceAccount{
				Name: "test-sa",
				UID:  "test-uid",
			},
		},
	}

	federationRef := &fedv1alpha1.FederationRef{
		Kind: "Kubernetes",
		Name: "test-federation",
	}

	// Call upsertIdentity
	err := server.upsertIdentity(
		ctx,
		authInfo,
		federationRef,
		"test-generator",
		"test-key",
		"Generator",
		"test-namespace",
		nil,
	)

	// Should succeed
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Create was called
	if !mockClient.createCalled {
		t.Error("expected Create to be called")
	}

	// Verify Update was not called
	if mockClient.updateCalled {
		t.Error("expected Update to not be called")
	}

	// Verify the created identity has the credential
	if mockClient.storedIdentity == nil {
		t.Fatal("expected identity to be stored")
	}

	if len(mockClient.storedIdentity.Spec.IssuedCredentials) != 1 {
		t.Errorf("expected 1 credential, got %d", len(mockClient.storedIdentity.Spec.IssuedCredentials))
	}

	// Verify the credential has correct source
	cred := mockClient.storedIdentity.Spec.IssuedCredentials[0]
	if cred.SourceRef.Name != "test-generator" {
		t.Errorf("expected source name 'test-generator', got '%s'", cred.SourceRef.Name)
	}
}

func TestUpsertIdentityUpdateWithNewCredential(t *testing.T) {
	// Test that upsertIdentity appends a new credential to an existing identity
	ctx := context.Background()

	// Create an existing identity with one credential
	existingIdentity := &fedv1alpha1.AuthorizedIdentity{
		Spec: fedv1alpha1.AuthorizedIdentitySpec{
			IssuedCredentials: []fedv1alpha1.IssuedCredential{
				{
					SourceRef: fedv1alpha1.SourceRef{
						Name: "existing-generator",
						Kind: "Generator",
					},
				},
			},
		},
	}

	mockClient := &mockClient{
		storedIdentity: existingIdentity,
	}

	reconciler := &externalsecrets.Reconciler{
		Client: mockClient,
	}

	server := &ServerHandler{
		reconciler: reconciler,
	}

	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: "test-provider",
		Subject:  "test-subject",
		KubeAttributes: &auth.KubeAttributes{
			Namespace: "test-ns",
			ServiceAccount: &auth.ServiceAccount{
				Name: "test-sa",
				UID:  "test-uid",
			},
		},
	}

	federationRef := &fedv1alpha1.FederationRef{
		Kind: "Kubernetes",
		Name: "test-federation",
	}

	// Call upsertIdentity with a different generator
	err := server.upsertIdentity(
		ctx,
		authInfo,
		federationRef,
		"new-generator",
		"test-key",
		"Generator",
		"test-namespace",
		nil,
	)

	// Should succeed
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Update was called, not Create
	if mockClient.createCalled {
		t.Error("expected Create to not be called")
	}

	if !mockClient.updateCalled {
		t.Error("expected Update to be called")
	}

	// Verify we now have 2 credentials
	if len(mockClient.storedIdentity.Spec.IssuedCredentials) != 2 {
		t.Errorf("expected 2 credentials, got %d", len(mockClient.storedIdentity.Spec.IssuedCredentials))
	}

	// Verify both credentials are present
	foundExisting := false
	foundNew := false
	for _, cred := range mockClient.storedIdentity.Spec.IssuedCredentials {
		if cred.SourceRef.Name == "existing-generator" {
			foundExisting = true
		}
		if cred.SourceRef.Name == "new-generator" {
			foundNew = true
		}
	}

	if !foundExisting {
		t.Error("existing credential was not preserved")
	}
	if !foundNew {
		t.Error("new credential was not added")
	}
}

func TestUpsertIdentityUpdateExistingCredential(t *testing.T) {
	// Test that upsertIdentity updates an existing credential without duplication
	// when the SAME workload re-requests the SAME credential
	ctx := context.Background()

	// Create an existing identity with one credential
	// Must match what buildSourceRef creates for Generator kind
	testNamespace := "test-namespace"
	testPodUID := "test-pod-uid-123"
	existingIdentity := &fedv1alpha1.AuthorizedIdentity{
		Spec: fedv1alpha1.AuthorizedIdentitySpec{
			IssuedCredentials: []fedv1alpha1.IssuedCredential{
				{
					SourceRef: fedv1alpha1.SourceRef{
						Name:       "test-generator",
						Kind:       "Generator",
						APIVersion: "generators.external-secrets.io/v1alpha1",
						Namespace:  &testNamespace,
					},
					RemoteRef: &fedv1alpha1.RemoteRef{
						RemoteKey: "same-key",
					},
					WorkloadBinding: &fedv1alpha1.WorkloadBinding{
						Kind:      "Pod",
						Name:      "test-pod",
						UID:       testPodUID,
						Namespace: "test-ns",
					},
				},
			},
		},
	}

	mockClient := &mockClient{
		storedIdentity: existingIdentity,
	}

	reconciler := &externalsecrets.Reconciler{
		Client: mockClient,
	}

	server := &ServerHandler{
		reconciler: reconciler,
	}

	// Same pod re-requesting
	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: "test-provider",
		Subject:  "test-subject",
		KubeAttributes: &auth.KubeAttributes{
			Namespace: "test-ns",
			ServiceAccount: &auth.ServiceAccount{
				Name: "test-sa",
				UID:  "test-uid",
			},
			Pod: &auth.PodInfo{
				Name: "test-pod", // Same pod name
				UID:  testPodUID, // Same pod UID
			},
		},
	}

	federationRef := &fedv1alpha1.FederationRef{
		Kind: "Kubernetes",
		Name: "test-federation",
	}

	// Call upsertIdentity with the same generator, key, and workload (should update, not append)
	err := server.upsertIdentity(
		ctx,
		authInfo,
		federationRef,
		"test-generator",
		"same-key",
		"Generator",
		"test-namespace",
		nil,
	)

	// Should succeed
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Verify Update was called
	if !mockClient.updateCalled {
		t.Error("expected Update to be called")
	}

	// Verify we still have only 1 credential (not duplicated)
	if len(mockClient.storedIdentity.Spec.IssuedCredentials) != 1 {
		t.Errorf("expected 1 credential (no duplication), got %d", len(mockClient.storedIdentity.Spec.IssuedCredentials))
	}

	// Verify the LastIssuedAt was updated (credential refreshed)
	cred := mockClient.storedIdentity.Spec.IssuedCredentials[0]
	if cred.WorkloadBinding == nil || cred.WorkloadBinding.Name != "test-pod" {
		t.Error("credential workload binding changed unexpectedly")
	}
}

func TestUpsertIdentityCreateError(t *testing.T) {
	// Test that upsertIdentity returns error when Create fails
	ctx := context.Background()

	createErr := errors.New("create failed")
	notFoundErr := apierrors.NewNotFound(schema.GroupResource{Group: "federation.external-secrets.io", Resource: "authorizedidentities"}, "test-identity")
	mockClient := &mockClient{
		getErr:    notFoundErr,
		createErr: createErr,
	}

	reconciler := &externalsecrets.Reconciler{
		Client: mockClient,
	}

	server := &ServerHandler{
		reconciler: reconciler,
	}

	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: "test-provider",
		Subject:  "test-subject",
		KubeAttributes: &auth.KubeAttributes{
			Namespace: "test-ns",
			ServiceAccount: &auth.ServiceAccount{
				Name: "test-sa",
				UID:  "test-uid",
			},
		},
	}

	federationRef := &fedv1alpha1.FederationRef{
		Kind: "Kubernetes",
		Name: "test-federation",
	}

	// Call upsertIdentity
	err := server.upsertIdentity(
		ctx,
		authInfo,
		federationRef,
		"test-generator",
		"test-key",
		"Generator",
		"test-namespace",
		nil,
	)

	// Should return the create error
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, createErr) {
		t.Errorf("expected create error, got: %v", err)
	}
}

func TestUpsertIdentityUpdateError(t *testing.T) {
	// Test that upsertIdentity returns error when Update fails
	ctx := context.Background()

	updateErr := errors.New("update failed")
	existingIdentity := &fedv1alpha1.AuthorizedIdentity{
		Spec: fedv1alpha1.AuthorizedIdentitySpec{
			IssuedCredentials: []fedv1alpha1.IssuedCredential{},
		},
	}

	mockClient := &mockClient{
		storedIdentity: existingIdentity,
		updateErr:      updateErr,
	}

	reconciler := &externalsecrets.Reconciler{
		Client: mockClient,
	}

	server := &ServerHandler{
		reconciler: reconciler,
	}

	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: "test-provider",
		Subject:  "test-subject",
		KubeAttributes: &auth.KubeAttributes{
			Namespace: "test-ns",
			ServiceAccount: &auth.ServiceAccount{
				Name: "test-sa",
				UID:  "test-uid",
			},
		},
	}

	federationRef := &fedv1alpha1.FederationRef{
		Kind: "Kubernetes",
		Name: "test-federation",
	}

	// Call upsertIdentity
	err := server.upsertIdentity(
		ctx,
		authInfo,
		federationRef,
		"test-generator",
		"test-key",
		"Generator",
		"test-namespace",
		nil,
	)

	// Should return the update error
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if !errors.Is(err, updateErr) {
		t.Errorf("expected update error, got: %v", err)
	}
}

func TestUpsertIdentityNilReconciler(t *testing.T) {
	// Test that upsertIdentity handles nil reconciler gracefully
	ctx := context.Background()

	server := &ServerHandler{
		reconciler: nil, // No reconciler
	}

	authInfo := &auth.AuthInfo{
		Method:   "oidc",
		Provider: "test-provider",
		Subject:  "test-subject",
		KubeAttributes: &auth.KubeAttributes{
			Namespace: "test-ns",
			ServiceAccount: &auth.ServiceAccount{
				Name: "test-sa",
				UID:  "test-uid",
			},
		},
	}

	federationRef := &fedv1alpha1.FederationRef{
		Kind: "Kubernetes",
		Name: "test-federation",
	}

	// Call upsertIdentity - should return nil without panicking
	err := server.upsertIdentity(
		ctx,
		authInfo,
		federationRef,
		"test-generator",
		"test-key",
		"Generator",
		"test-namespace",
		nil,
	)

	// Should succeed (early return)
	if err != nil {
		t.Errorf("expected no error with nil reconciler, got: %v", err)
	}
}
