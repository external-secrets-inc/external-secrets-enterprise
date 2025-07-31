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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=RSA
type SSHKeyType string

const (
	SSHKeyTypeRSA SSHKeyType = "RSA"
)

// SSHSpec controls the behavior of the password generator.
type SSHSpec struct {
	// KeyType specifies the SSH key type to be generated.
	// Currently only RSA is supported.
	// +kubebuilder:default="RSA"
	KeyType SSHKeyType `json:"keyType,omitempty"`

	// RSAConfig specifies the configuration of the RSA key to be generated.
	RSAConfig RSASpec `json:"rsaConfig,omitempty"`
}

// RSASpec controls the behavior of the password generator.
type RSASpec struct {
	// Bit size of the RSA Key to be generated.
	// Defaults to 4096
	// +kubebuilder:default=4096
	Bits int `json:"bits"`
}

// SSH generates a random ssh based on the
// configuration parameters in spec.
// You can specify the length, characterset and other attributes.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type SSH struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SSHSpec                     `json:"spec,omitempty"`
	Status genv1alpha1.GeneratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SSHList contains a list of ExternalSecret resources.
type SSHList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SSH `json:"items"`
}
