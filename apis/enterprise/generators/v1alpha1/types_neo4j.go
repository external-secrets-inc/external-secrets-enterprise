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

// Neo4jSpec controls the behavior of the neo4j generator.
type Neo4jSpec struct {
	// Database is the name of the database to connect to.
	// If not specified, the "neo4j" database will be used.
	// +kubebuilder:default=neo4j
	Database string `json:"database"`
	// Auth contains the credentials or auth configuration
	Auth Neo4jAuth `json:"auth"`
	// User is the data of the user to be created.
	User *Neo4jUser `json:"user,omitempty"`
	// If the neo4j instance is running in enterprise mode.
	// +kubebuilder:default=false
	Enterprise bool `json:"enterprise,omitempty"`
}

type Neo4jAuth struct {
	// URI is the connection URI for the Neo4j database.
	// Example: bolt://neo4j.default.svc.cluster.local:7687
	URI string `json:"uri"`
	// Basic auth credentials used to authenticate against the Neo4j instance.
	// +optional
	Basic *Neo4jBasicAuth `json:"basic,omitempty"`
	// Bearer auth token used to authenticate against the Neo4j instance.
	// +optional
	Bearer *Neo4jBearerAuth `json:"bearer,omitempty"`
}

type Neo4jBasicAuth struct {
	// A basic auth username used to authenticate against the Neo4j instance.
	Username string `json:"username"`
	// A basic auth password used to authenticate against the Neo4j instance.
	Password esmeta.SecretKeySelector `json:"password"`
}

type Neo4jBearerAuth struct {
	// A bearer auth token used to authenticate against the Neo4j instance.
	Token esmeta.SecretKeySelector `json:"token"`
}

type Neo4jAuthProvider string

const (
	Neo4jAuthProviderNative Neo4jAuthProvider = "native"
)

type Neo4jUser struct {
	// The name of the user to be created.
	// Must not contain a dash ("-") character.
	// +kubebuilder:validation:Pattern=`^[^-\s]+$`
	User string `json:"user"`
	// SuffixSize define the size of the random suffix added after the defined username.
	// If not specified, a random suffix of size 8 will be used.
	// If set to 0, no suffix will be added.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=8
	SuffixSize *int `json:"suffixSize,omitempty"`
	// The roles to be assigned to the user (Neo4j Enterprise only).
	// See https://neo4j.com/docs/operations-manual/current/authentication-authorization/built-in-roles/
	// for a list of built-in roles.
	// If contains non-existing roles, they will be created as copy of "PUBLIC" role.
	// If empty, the user will be created with no role.
	Roles []string `json:"roles"`
	// The home database of the user (Neo4j Enterprise only).
	Home *string `json:"home,omitempty"`
	// The auth provider to be used for the user (Neo4j Enterprise only).
	// Currently only "native" is supported.
	// "native" provider is used for Neo4j Community
	// +kubebuilder:validation:Enum=native
	// +kubebuilder:default=native
	Provider Neo4jAuthProvider `json:"provider"`
}

type Neo4jUserState struct {
	User string `json:"user"`
}

// Neo4j generates a random neo4j based on the
// configuration parameters in spec.
// You can specify the length, characterset and other attributes.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type Neo4j struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Neo4jSpec                   `json:"spec,omitempty"`
	Status genv1alpha1.GeneratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Neo4jList contains a list of ExternalSecret resources.
type Neo4jList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Neo4j `json:"items"`
}
