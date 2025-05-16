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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type AuthorizationSpec struct {
	FederationRef              FederationRef      `json:"federationRef"`
	Subject                    FederationSubject  `json:"subject"`
	AllowedClusterSecretStores []string           `json:"allowedClusterSecretStores"`
	AllowedGenerators          []AllowedGenerator `json:"allowedGenerators"`
}
type AllowedGenerator struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
}
type FederationRef struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

type FederationSubject struct {
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
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
