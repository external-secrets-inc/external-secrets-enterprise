// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	api "github.com/external-secrets/external-secrets/apis/federation/v1alpha1"
)

var authorizationStore sync.Map
var federationStore sync.Map

func init() {
	authorizationStore = sync.Map{}
	federationStore = sync.Map{}
}

func AddStore(name api.FederationRef, provider api.FederationProvider) {
	federationStore.Store(name, provider)
}

func GetStore(name api.FederationRef) api.FederationProvider {
	s, ok := federationStore.Load(name)
	if !ok {
		return nil
	}
	return s.(api.FederationProvider)
}

func Add(issuer string, ref *api.AuthorizationSpec) {
	values := []*api.AuthorizationSpec{ref}

	if v, ok := authorizationStore.Load(issuer); ok {
		values = append(values, v.([]*api.AuthorizationSpec)...)
	}
	authorizationStore.Store(issuer, values)
}

func Remove(issuer string, ref *api.AuthorizationSpec) {
	authorizationStore.Delete(issuer)
}

func Get(issuer string) []*api.AuthorizationSpec {
	r, ok := authorizationStore.Load(issuer)
	if !ok {
		return nil
	}
	return r.([]*api.AuthorizationSpec)
}

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
