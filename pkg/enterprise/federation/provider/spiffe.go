// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

package provider

import (
	"context"
)

type SpiffeProvider struct {
	TrustDomain string
}

func NewSpiffeProvider(trustDomain string) *SpiffeProvider {
	return &SpiffeProvider{
		TrustDomain: trustDomain,
	}
}

func (k *SpiffeProvider) GetJWKS(ctx context.Context, token, issuer string, caCrt []byte) (map[string]map[string]string, error) {
	return nil, nil
}

// CheckIdentityExists checks if a SPIFFE identity still exists.
// For SPIFFE federation, identity lifecycle is managed through WorkloadBinding and mTLS certificate validation,
// so this always returns true (identity check happens via workload lifecycle).
func (k *SpiffeProvider) CheckIdentityExists(ctx context.Context, subject string) (bool, error) {
	// SPIFFE federation uses WorkloadBinding and certificate validation for lifecycle management
	// This method is not used for SPIFFE identities
	return true, nil
}
