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
)

type KafkaAuthMechanismEnum string

const (
	KafkaAuthMechanismPlain    KafkaAuthMechanismEnum = "PLAIN"
	KafkaAuthMechanismScram512 KafkaAuthMechanismEnum = "SCRAM-SHA-512"
	KafkaAuthMechanismScram256 KafkaAuthMechanismEnum = "SCRAM-SHA-256"
)

// KafkaSpec controls the behavior of the kafka generator.
type KafkaSpec struct {
	// Addresses of the kafka cluster(s) (or specific broker) that the client
	// will be sending requests to.
	Addresses []string `json:"addresses"`
	// Auth contains the credentials or auth configuration
	Auth KafkaAuth `json:"auth"`
	// User is the data of the user to be created.
	User *KafkaUser `json:"user,omitempty"`
}

type KafkaAuth struct {
	// A basic auth username used to authenticate against the Kafka instance.
	Username string `json:"username"`
	// A basic auth password used to authenticate against the Kafka instance.
	Password SecretKeySelector `json:"password"`
	// Mechanism define the SASL mechanism to use (Optional.
	// Accepted values: PLAIN, SCRAM-SHA-512, SCRAM-SHA-256
	// +kubebuilder:validation:Enum=PLAIN;SCRAM-SHA-512;SCRAM-SHA-256
	// +optional
	Mechanism *KafkaAuthMechanismEnum `json:"mechanism,omitempty"`
	// TLSConfig holds TLS configuration options for connecting securely.
	// +optional
	TLSConfig *KafkaTLSConfig `json:"TLSConfig,omitempty"`
}

type KafkaTLSConfig struct {
	// CACert is the reference to the CA certificate used to verify the server certificate.
	// +optional
	CACert *SecretKeySelector `json:"caCert,omitempty"`
	// ClientCert is the reference to the client certificate used for mutual TLS.
	// +optional
	ClientCert *SecretKeySelector `json:"clientCert,omitempty"`
	// ClientKey is the reference to the private key used for mutual TLS.
	// +optional
	ClientKey *SecretKeySelector `json:"clientKey,omitempty"`
}

type KafkaUser struct {
	// The username of the user to be created.
	Username string `json:"username"`
	// SuffixSize define the size of the random suffix added after the defined username.
	// If not specified, a random suffix of size 8 will be used.
	// If set to 0, no suffix will be added.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=8
	SuffixSize *int `json:"suffixSize,omitempty"`
	// Permissions is the list of ACLs that will be assigned to this user.
	Permissions []KafkaUserPermissions `json:"permissions,omitempty"`
}

type KafkaUserPermissions struct {
	// ResourceType is the type of Kafka resource.
	// Accepted values: topic, group, cluster, transactional_id, any.
	// +kubebuilder:validation:Enum=topic;group;cluster;transactional_id;any
	ResourceType string `json:"resourceType"`

	// ResourceName is the name of the Kafka resource (e.g., topic name or group ID).
	// +kubebuilder:validation:MinLength=1
	ResourceName string `json:"resourceName"`

	// ResourcePatternType defines the match type for the resource name.
	// Accepted values: literal, prefixed, any.
	// +kubebuilder:validation:Enum=literal;prefixed;any
	ResourcePatternType string `json:"resourcePatternType"`

	// Host is the IP or hostname from which access applies.
	// If empty, the client address will be used.
	// +optional
	Host string `json:"host,omitempty"`

	// Operation is the Kafka action being granted or denied.
	// Accepted values: read, write, create, delete, describe, alter, all, cluster_action,
	// describe_configs, alter_configs, idempotent_write, any.
	// +kubebuilder:validation:Enum=read;write;create;delete;describe;alter;all;cluster_action;describe_configs;alter_configs;idempotent_write;any
	Operation string `json:"operation"`

	// PermissionType specifies whether the permission is allowed or denied.
	// Accepted values: allow, deny, any.
	// +kubebuilder:validation:Enum=allow;deny;any
	PermissionType string `json:"permissionType"`
}

type KafkaUserState struct {
	Username string `json:"username,omitempty"`
}

// Kafka generates a random kafka based on the
// configuration parameters in spec.
// You can specify the length, characterset and other attributes.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KafkaSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// KafkaList contains a list of ExternalSecret resources.
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kafka `json:"items"`
}
