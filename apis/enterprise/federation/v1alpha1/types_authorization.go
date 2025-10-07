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
package v1alpha1

import (
	"errors"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:XValidation:rule="(has(self.subject.spiffe) && !has(self.subject.oidc)) || (!has(self.subject.spiffe) && has(self.subject.oidc))",message="spiffe or subject must be set"
type AuthorizationSpec struct {
	FederationRef FederationRef `json:"federationRef"`

	// +kubebuilder:validation:Required
	Subject *FederationSubject `json:"subject,omitempty"`

	// Which ClusterSecretStores can this subject request
	AllowedClusterSecretStores []string `json:"allowedClusterSecretStores"`
	// Which Generators namespaces can this subject request
	AllowedGenerators []AllowedGenerator `json:"allowedGenerators"`
	// Which GeneratorState namespaces can this subject delete
	AllowedGeneratorStates []AllowedGeneratorState `json:"allowedGeneratorStates"`
}

func (a *AuthorizationSpec) RequiresTLS() bool {
	switch {
	case a.Subject.Spiffe != nil:
		return true
	case a.Subject.OIDC != nil:
		return false
	default:
		return false
	}
}

func (a *AuthorizationSpec) Principal() (string, error) {
	switch {
	case a.Subject.Spiffe != nil:
		return a.Subject.Spiffe.SpiffeID, nil
	case a.Subject.OIDC != nil:
		return a.Subject.OIDC.Subject, nil
	default:
		return "", errors.New("no subject configured (choose spiffe or subject)")
	}
}

func (a *AuthorizationSpec) Authority() (string, error) {
	switch {
	case a.Subject.Spiffe != nil:
		spiffeID, err := spiffeid.FromString(a.Subject.Spiffe.SpiffeID)
		if err != nil {
			return "", err
		}
		return spiffeID.TrustDomain().Name(), nil
	case a.Subject.OIDC != nil:
		return a.Subject.OIDC.Issuer, nil
	default:
		return "", errors.New("no issuer configured (choose spiffe or subject)")
	}
}

type AllowedGeneratorState struct {
	Namespace string `json:"namespace"`
}
type AllowedGenerator struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	// +kubebuilder:default="generators.external-secrets.io/v1alpha1"
	ApiVersion string `json:"apiVersion"`
}
type FederationRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type FederationSubject struct {
	// +kubebuilder:validation:Optional
	Spiffe *FederationSpiffe `json:"spiffe"`
	// +kubebuilder:validation:Optional
	OIDC *FederationOIDC `json:"oidc"`
}

type FederationOIDC struct {
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
}

type FederationSpiffe struct {
	SpiffeID string `json:"spiffeID"`
}

// Todo - This should be the permission for secretStores instead of slice of string
// type FederationPermission struct {
// 	Name     string                `json:"name"`
// 	Selector *metav1.LabelSelector `json:"selector,omitempty"`
// }

// Authorization is the schema to control authorization over ClusterSecretStores.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Cluster,categories={external-secrets, external-secrets-federation}
type Authorization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AuthorizationSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// AuthorizationList contains a list of Authorization resources.
type AuthorizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Authorization `json:"items"`
}
