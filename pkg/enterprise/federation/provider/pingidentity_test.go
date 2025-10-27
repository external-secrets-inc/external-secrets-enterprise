// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPingIdentityProvider(t *testing.T) {
	tests := []struct {
		name          string
		region        string
		environmentID string
	}{
		{
			name:          "with region com",
			region:        "com",
			environmentID: "12345678-1234-1234-1234-123456789abc",
		},
		{
			name:          "with region eu",
			region:        "eu",
			environmentID: "87654321-4321-4321-4321-cba987654321",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewPingIdentityProvider(tt.region, tt.environmentID)

			assert.Equal(t, tt.region, provider.Region)
			assert.Equal(t, tt.environmentID, provider.EnvironmentID)
			assert.NotNil(t, provider.httpClient)
			assert.NotNil(t, provider.jwksCache)
			assert.Equal(t, defaultPingIdentityJWKSCacheTTL, provider.cacheTTL)
		})
	}
}

func TestPingIdentityProvider_GetJWKS(t *testing.T) {
	tests := []struct {
		name                  string
		mockDiscoveryResponse interface{}
		mockDiscoveryStatus   int
		mockJWKSResponse      interface{}
		mockJWKSStatus        int
		expectError           bool
		errorContains         string
		expectedKeys          int
	}{
		{
			name:                "successful JWKS fetch",
			mockDiscoveryStatus: http.StatusOK,
			mockDiscoveryResponse: map[string]interface{}{
				"jwks_uri": "JWKS_URL_PLACEHOLDER",
			},
			mockJWKSStatus: http.StatusOK,
			mockJWKSResponse: map[string]interface{}{
				"keys": []map[string]string{
					{
						"kid": "key1",
						"kty": "RSA",
						"n":   "test-modulus",
						"e":   "AQAB",
					},
					{
						"kid": "key2",
						"kty": "RSA",
						"n":   "test-modulus-2",
						"e":   "AQAB",
					},
				},
			},
			expectedKeys: 2,
		},
		{
			name:                "empty JWKS response",
			mockDiscoveryStatus: http.StatusOK,
			mockDiscoveryResponse: map[string]interface{}{
				"jwks_uri": "JWKS_URL_PLACEHOLDER",
			},
			mockJWKSStatus: http.StatusOK,
			mockJWKSResponse: map[string]interface{}{
				"keys": []map[string]string{},
			},
			expectError:   true,
			errorContains: "no valid keys found",
		},
		{
			name:                "JWKS keys without kid",
			mockDiscoveryStatus: http.StatusOK,
			mockDiscoveryResponse: map[string]interface{}{
				"jwks_uri": "JWKS_URL_PLACEHOLDER",
			},
			mockJWKSStatus: http.StatusOK,
			mockJWKSResponse: map[string]interface{}{
				"keys": []map[string]string{
					{
						"kty": "RSA",
						"n":   "test-modulus",
						"e":   "AQAB",
					},
				},
			},
			expectError:   true,
			errorContains: "no valid keys found",
		},
		{
			name:                "HTTP error status from JWKS",
			mockDiscoveryStatus: http.StatusOK,
			mockDiscoveryResponse: map[string]interface{}{
				"jwks_uri": "JWKS_URL_PLACEHOLDER",
			},
			mockJWKSStatus:   http.StatusUnauthorized,
			mockJWKSResponse: map[string]string{"error": "unauthorized"},
			expectError:      true,
			errorContains:    "status 401",
		},
		{
			name:                "invalid JSON in JWKS response",
			mockDiscoveryStatus: http.StatusOK,
			mockDiscoveryResponse: map[string]interface{}{
				"jwks_uri": "JWKS_URL_PLACEHOLDER",
			},
			mockJWKSStatus:   http.StatusOK,
			mockJWKSResponse: "invalid json",
			expectError:      true,
			errorContains:    "failed to parse JWKS response",
		},
		{
			name:                "discovery endpoint error",
			mockDiscoveryStatus: http.StatusNotFound,
			mockDiscoveryResponse: map[string]string{
				"error": "not found",
			},
			expectError:   true,
			errorContains: "discovery endpoint returned status 404",
		},
		{
			name:                "discovery missing jwks_uri",
			mockDiscoveryStatus: http.StatusOK,
			mockDiscoveryResponse: map[string]interface{}{
				"issuer": "https://auth.pingone.com/env-id",
			},
			expectError:   true,
			errorContains: "discovery document missing jwks_uri field",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock servers
			var jwksURL string
			jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockJWKSStatus)
				if str, ok := tt.mockJWKSResponse.(string); ok {
					_, _ = w.Write([]byte(str))
				} else {
					_ = json.NewEncoder(w).Encode(tt.mockJWKSResponse)
				}
			}))
			defer jwksServer.Close()
		jwksURL = jwksServer.URL

		discoveryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify discovery path
			assert.Equal(t, "/.well-known/openid-configuration", r.URL.Path)

			w.WriteHeader(tt.mockDiscoveryStatus)

			// Replace placeholder with actual JWKS URL
			switch response := tt.mockDiscoveryResponse.(type) {
			case map[string]interface{}:
				if response["jwks_uri"] == "JWKS_URL_PLACEHOLDER" {
					response["jwks_uri"] = jwksURL
				}
				_ = json.NewEncoder(w).Encode(response)
			case string:
				_, _ = w.Write([]byte(response))
			default:
				_ = json.NewEncoder(w).Encode(tt.mockDiscoveryResponse)
			}
		}))
		defer discoveryServer.Close()

		// Create provider and override the discovery URL for testing
		provider := NewPingIdentityProvider("com", "test-env-id")
		provider.discoveryBaseURL = discoveryServer.URL

		ctx := context.Background()

			jwks, err := provider.GetJWKS(ctx, "", "", nil)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Len(t, jwks, tt.expectedKeys)
			}
		})
	}
}

func TestPingIdentityProvider_GetJWKS_Caching(t *testing.T) {
	discoveryRequestCount := 0
	jwksRequestCount := 0

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwksRequestCount++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"keys": []map[string]string{
				{
					"kid": "key1",
					"kty": "RSA",
					"n":   "test-modulus",
					"e":   "AQAB",
				},
			},
		})
	}))
	defer jwksServer.Close()

	discoveryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		discoveryRequestCount++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jwks_uri": jwksServer.URL,
		})
	}))
	defer discoveryServer.Close()

	provider := NewPingIdentityProvider("com", "test-env")
	provider.discoveryBaseURL = discoveryServer.URL
	provider.cacheTTL = 100 * time.Millisecond // Short TTL for testing

	ctx := context.Background()

	// First request - should hit both discovery and JWKS
	jwks1, err := provider.GetJWKS(ctx, "", "", nil)
	require.NoError(t, err)
	assert.Len(t, jwks1, 1)
	assert.Equal(t, 1, discoveryRequestCount)
	assert.Equal(t, 1, jwksRequestCount)

	// Second request - should use cache
	jwks2, err := provider.GetJWKS(ctx, "", "", nil)
	require.NoError(t, err)
	assert.Len(t, jwks2, 1)
	assert.Equal(t, 1, discoveryRequestCount, "should not make another discovery request due to cache")
	assert.Equal(t, 1, jwksRequestCount, "should not make another JWKS request due to cache")

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third request - should hit JWKS again (but not discovery, as jwksURL is cached)
	jwks3, err := provider.GetJWKS(ctx, "", "", nil)
	require.NoError(t, err)
	assert.Len(t, jwks3, 1)
	assert.Equal(t, 1, discoveryRequestCount, "should not make another discovery request as jwksURL is cached")
	assert.Equal(t, 2, jwksRequestCount, "should make another JWKS request after cache expiry")
}

func TestPingIdentityProvider_GetJWKS_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jwks_uri": "http://example.com/jwks",
		})
	}))
	defer server.Close()

	provider := NewPingIdentityProvider("com", "test-env")
	provider.discoveryBaseURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := provider.GetJWKS(ctx, "", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestPingIdentityProvider_CheckIdentityExists(t *testing.T) {
	provider := NewPingIdentityProvider("com", "test-env-id")
	ctx := context.Background()

	// Should always return true as it's not implemented
	exists, err := provider.CheckIdentityExists(ctx, "some-client-id")
	require.NoError(t, err)
	assert.True(t, exists)
}
