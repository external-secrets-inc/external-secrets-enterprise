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
