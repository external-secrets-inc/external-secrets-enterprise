// Package store implements the authorization store.
// Copyright External Secrets Inc.
// All Rights Reserved.
package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	api "github.com/external-secrets/external-secrets/apis/enterprise/federation/v1alpha1"
)

var authorizationStore sync.Map
var federationStore sync.Map

func init() {
	authorizationStore = sync.Map{}
	federationStore = sync.Map{}
}

// AddStore adds a new provider to the federation store.
func AddStore(name api.FederationRef, provider api.FederationProvider) {
	federationStore.Store(name, provider)
}

// GetStore returns the provider for the given federation ref.
func GetStore(name api.FederationRef) api.FederationProvider {
	s, ok := federationStore.Load(name)
	if !ok {
		return nil
	}
	return s.(api.FederationProvider)
}

// Add adds a new AuthorizationSpec to the authorization store.
func Add(issuer string, ref *api.AuthorizationSpec) {
	values := []*api.AuthorizationSpec{ref}

	if v, ok := authorizationStore.Load(issuer); ok {
		values = append(values, v.([]*api.AuthorizationSpec)...)
	}
	authorizationStore.Store(issuer, values)
}

// Remove removes the AuthorizationSpec for the given issuer.
func Remove(issuer string, _ *api.AuthorizationSpec) {
	authorizationStore.Delete(issuer)
}

// Get returns the AuthorizationSpec for the given issuer.
func Get(issuer string) []*api.AuthorizationSpec {
	r, ok := authorizationStore.Load(issuer)
	if !ok {
		return nil
	}
	return r.([]*api.AuthorizationSpec)
}

// GetJWKS returns the JWKS for the given issuer.
func GetJWKS(ctx context.Context, specs []*api.AuthorizationSpec, token, issuer string, caCrt []byte) (map[string]map[string]string, error) {
	for _, spec := range specs {
		providerRef := spec.FederationRef
		provider := GetStore(providerRef)
		if provider == nil {
			return nil, errors.New("no provider found")
		}
		jwks, err := provider.GetJWKS(ctx, token, issuer, caCrt)
		if err != nil {
			fmt.Println(err)
			// Not This One, go to next
			continue
		}
		return jwks, nil
	}
	return nil, errors.New("no jwks found")
}

// CheckIfExists checks if the identity still exists in the identity provider.
// It looks up the provider by federationRef and calls its CheckIdentityExists method.
// Returns true if exists, false if deleted, or error if check failed.
func CheckIfExists(ctx context.Context, federationRef api.FederationRef, subject string) (bool, error) {
	provider := GetStore(federationRef)
	if provider == nil {
		return false, fmt.Errorf("no provider found for federation ref: %s/%s", federationRef.Kind, federationRef.Name)
	}

	return provider.CheckIdentityExists(ctx, subject)
}
