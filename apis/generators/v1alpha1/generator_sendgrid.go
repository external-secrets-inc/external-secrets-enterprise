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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type SendgridTokenSpec struct {
	// +kubebuilder:default=global
	DataResidency string       `json:"dataResidency,omitempty"`
	Scopes        []string     `json:"scopes,omitempty"`
	Auth          SendgridAuth `json:"auth,omitempty"`
}

type SendgridAuth struct {
	SecretRef *SendgridAuthSecretRef `json:"secretRef,omitempty"`
}

type SendgridAuthSecretRef struct {
	APIKey esmeta.SecretKeySelector `json:"apiKeySecretRef,omitempty"`
}

// SendgridAuthorizationToken generates sendgrid api keys
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={sendgridauthorizationtoken},shortName=sendgridauthorizationtoken
type SendgridAuthorizationToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SendgridTokenSpec `json:"spec,omitempty"`
	Status GeneratorStatus   `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type SendgridAuthorizationTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SendgridAuthorizationToken `json:"items"`
}
