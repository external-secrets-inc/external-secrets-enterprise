// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	// Default cache TTL for JWKS (1 hour).
	defaultPingIdentityJWKSCacheTTL = 1 * time.Hour
)

type PingIdentityProvider struct {
	Region        string
	EnvironmentID string
	httpClient    *http.Client
	jwksCache     map[string]map[string]string
	cacheMutex    sync.RWMutex
	lastFetch     time.Time
	cacheTTL      time.Duration
	jwksURL       string // Cached JWKS URL from discovery
	// discoveryBaseURL is used for testing to override the discovery endpoint
	// If empty, uses the standard PingOne URL format
	discoveryBaseURL string
}

func NewPingIdentityProvider(region, environmentID string) *PingIdentityProvider {
	return &PingIdentityProvider{
		Region:        region,
		EnvironmentID: environmentID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		jwksCache: make(map[string]map[string]string),
		cacheTTL:  defaultPingIdentityJWKSCacheTTL,
	}
}

// GetJWKS fetches the JSON Web Key Set from PingOne's public endpoint.
// The token, issuer, and caCrt parameters are not used for PingOne since the
// JWKS endpoint is publicly accessible over standard HTTPS.
func (p *PingIdentityProvider) GetJWKS(ctx context.Context, token, issuer string, caCrt []byte) (map[string]map[string]string, error) {
	p.cacheMutex.RLock()
	// Check if cache is still valid
	if time.Since(p.lastFetch) < p.cacheTTL && len(p.jwksCache) > 0 {
		cachedJWKS := p.jwksCache
		p.cacheMutex.RUnlock()
		return cachedJWKS, nil
	}
	p.cacheMutex.RUnlock()

	// Fetch fresh JWKS
	return p.fetchAndCacheJWKS(ctx)
}

func (p *PingIdentityProvider) fetchAndCacheJWKS(ctx context.Context) (map[string]map[string]string, error) {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	// Double-check after acquiring write lock
	if time.Since(p.lastFetch) < p.cacheTTL && len(p.jwksCache) > 0 {
		return p.jwksCache, nil
	}

	// If we don't have the JWKS URL yet, fetch it from discovery
	if p.jwksURL == "" {
		if err := p.fetchJWKSURLFromDiscovery(ctx); err != nil {
			return nil, fmt.Errorf("failed to fetch JWKS URL from discovery: %w", err)
		}
	}

	// Fetch JWKS from the discovered URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWKS request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from PingOne: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("JWKS endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWKS response: %w", err)
	}

	var jwksResponse struct {
		Keys []map[string]interface{} `json:"keys"`
	}

	if err := json.Unmarshal(body, &jwksResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JWKS response: %w (body: %s)", err, string(body))
	}

	// Convert to map[kid]key format, converting values to strings
	jwksMap := make(map[string]map[string]string)
	for _, key := range jwksResponse.Keys {
		kidInterface, ok := key["kid"]
		if !ok {
			continue
		}
		kid, ok := kidInterface.(string)
		if !ok {
			continue
		}

		// Convert interface{} values to strings (skip arrays like x5c)
		stringKey := make(map[string]string)
		for k, v := range key {
			if strVal, ok := v.(string); ok {
				stringKey[k] = strVal
			}
			// Skip arrays and other non-string types
		}
		jwksMap[kid] = stringKey
	}

	if len(jwksMap) == 0 {
		return nil, fmt.Errorf("no valid keys found in JWKS response")
	}

	// Update cache
	p.jwksCache = jwksMap
	p.lastFetch = time.Now()

	return jwksMap, nil
}

// fetchJWKSURLFromDiscovery fetches the JWKS URI from PingOne's OIDC discovery endpoint.
func (p *PingIdentityProvider) fetchJWKSURLFromDiscovery(ctx context.Context) error {
	// Construct discovery URL: https://auth.pingone.{region}/{envID}/as/.well-known/openid-configuration
	var discoveryURL string
	if p.discoveryBaseURL != "" {
		// Use override for testing
		discoveryURL = p.discoveryBaseURL + "/.well-known/openid-configuration"
	} else {
		// Use standard PingOne URL format (note the /as path component)
		discoveryURL = fmt.Sprintf("https://auth.pingone.%s/%s/as/.well-known/openid-configuration", p.Region, p.EnvironmentID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create discovery request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch discovery document: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discovery endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read discovery response: %w", err)
	}

	var discoveryDoc struct {
		JwksURI string `json:"jwks_uri"`
	}

	if err := json.Unmarshal(body, &discoveryDoc); err != nil {
		return fmt.Errorf("failed to parse discovery document: %w", err)
	}

	if discoveryDoc.JwksURI == "" {
		return fmt.Errorf("discovery document missing jwks_uri field")
	}

	p.jwksURL = discoveryDoc.JwksURI
	return nil
}

// CheckIdentityExists is not implemented for PingOne as there's no readily available
// Management API equivalent to check if applications/clients still exist.
// Returns true (assume exists) to avoid breaking existing functionality.
func (p *PingIdentityProvider) CheckIdentityExists(ctx context.Context, subject string) (bool, error) {
	// No Management API integration - assume identity exists
	return true, nil
}
