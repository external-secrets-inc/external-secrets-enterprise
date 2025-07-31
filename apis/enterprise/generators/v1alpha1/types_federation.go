/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package v1alpha1

import (
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FederationSpec defines the configuration for the federation generator.
type FederationSpec struct {
	// Server specifies the federation server configuration
	Server FederationServer `json:"server"`

	// Generator specifies the target generator to use
	Generator FederationGeneratorRef `json:"generator"`

	// Auth specifies the authentication configuration
	Auth FederationAuthKubernetes `json:"auth"`
}

// FederationServer defines the federation server configuration.
type FederationServer struct {
	// URL is the URL of the federation server
	URL string `json:"url"`
}

// FederationGeneratorRef defines the target generator.
type FederationGeneratorRef struct {
	// Namespace is the namespace of the generator
	Namespace string `json:"namespace"`

	// Kind is the kind of the generator
	Kind string `json:"kind"`

	// Name is the name of the generator
	Name string `json:"name"`
}

// FederationAuthKubernetes defines the authentication configuration.
type FederationAuthKubernetes struct {
	// TokenSecretRef references a secret containing the auth token
	// +optional
	TokenSecretRef *esmeta.SecretKeySelector `json:"tokenSecretRef,omitempty"`

	// CACertSecretRef references a secret containing the CA certificate
	// +optional
	CACertSecretRef *esmeta.SecretKeySelector `json:"caCertSecretRef,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type Federation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FederationSpec              `json:"spec"`
	Status genv1alpha1.GeneratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type FederationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Federation `json:"items"`
}
