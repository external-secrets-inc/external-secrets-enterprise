// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package auth

import "net/http"

// AuthInfo contains information about the authenticated user.
type AuthInfo struct {
	// Method is the authentication method used, either "oidc" or "spiffe".
	Method string `json:"method"`
	// Provider is the provider of the authentication, either an OIDC issuer URL or a SPIFFE trust domain.
	Provider string `json:"provider"`
	// Subject is the subject of the authentication, either the OIDC subject or the SPIFFE ID.
	Subject string `json:"subject"`
	// KubeAttributes contains information about the user's Kubernetes context.
	KubeAttributes *KubeAttributes `json:"kubeAttributes"`
}

// KubeAttributes contains information about the user's Kubernetes context.
type KubeAttributes struct {
	// Namespace is the namespace of the user's context.
	Namespace string `json:"namespace"`
	// ServiceAccount is the user's service account.
	ServiceAccount *ServiceAccount `json:"serviceaccount"`
	// Pod is the user's pod, if any.
	Pod *PodInfo `json:"pod,omitempty"`
}

type ServiceAccount struct {
	// Name is the name of the service account.
	Name string `json:"name"`
	// UID is the UID of the service account.
	UID string `json:"uid"`
}

type PodInfo struct {
	// Name is the name of the pod.
	Name string `json:"name"`
	// UID is the UID of the pod.
	UID string `json:"uid"`
}

// Authenticator is the interface that an authentication implementation must
// implement.
type Authenticator interface {
	// Authenticate authenticates the given request and returns an AuthInfo
	// if the authentication is successful. The returned AuthInfo contains
	// information about the authenticated user.
	Authenticate(r *http.Request) (*AuthInfo, error)
}

// Registry is the registry of authenticators, mapping names to
// implementations.
var Registry = make(map[string]Authenticator)

// Register registers an authenticator implementation with the given name.
func Register(name string, a Authenticator) {
	Registry[name] = a
}
