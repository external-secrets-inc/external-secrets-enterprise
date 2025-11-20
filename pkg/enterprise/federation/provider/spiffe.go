// Package provider implements the federation provider.
// Copyright External Secrets Inc.
// All Rights Reserved.
package provider

import (
	"context"
)

// SpiffeProvider implements the SPIFFE provider for federation.
type SpiffeProvider struct {
	TrustDomain string
}

// NewSpiffeProvider creates a new SPIFFE provider.
func NewSpiffeProvider(trustDomain string) *SpiffeProvider {
	return &SpiffeProvider{
		TrustDomain: trustDomain,
	}
}

// GetJWKS returns the JWKS for the SPIFFE provider.
func (k *SpiffeProvider) GetJWKS(_ context.Context, _, _ string, _ []byte) (map[string]map[string]string, error) {
	return nil, nil
}

// CheckIdentityExists checks if a SPIFFE identity still exists.
// For SPIFFE federation, identity lifecycle is managed through WorkloadBinding and mTLS certificate validation,
// so this always returns true (identity check happens via workload lifecycle).
func (k *SpiffeProvider) CheckIdentityExists(_ context.Context, _ string) (bool, error) {
	// SPIFFE federation uses WorkloadBinding and certificate validation for lifecycle management
	// This method is not used for SPIFFE identities
	return true, nil
}
