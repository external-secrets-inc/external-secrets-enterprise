// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package store

import (
	"context"
	"errors"
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
			// Not This One, go to next
			continue
		}
		return jwks, nil
	}
	return nil, errors.New("no jwks found")
}
