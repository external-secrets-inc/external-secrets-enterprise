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

type PostgreSqlCleanupPolicy struct {
	genv1alpha1.CleanupPolicy `json:",inline"`

	// ActivityTrackingInterval is the cron expression to run the user activity tracking
	// +optional
	// +kubebuilder:default="2s"
	ActivityTrackingInterval metav1.Duration `json:"activityTrackingInterval,omitempty"`
}

// PostgreSqlSpec controls the behavior of the postgreSQL generator.
type PostgreSqlSpec struct {
	// Database is the name of the database to connect to.
	// If not specified, the "postgres" database will be used.
	// +kubebuilder:default=postgres
	Database string `json:"database"`
	// Host is the server where the database is hosted.
	Host string `json:"host"`
	// Port is the port of the database to connect to.
	// If not specified, the "5432" port will be used.
	// +kubebuilder:validation:Pattern=`^([0-9]{1,5}|[0-9]{1,5}\/[0-9]{1,5})$`
	// +kubebuilder:default="5432"
	Port string `json:"port"`
	// Auth contains the credentials or auth configuration
	Auth PostgreSqlAuth `json:"auth"`
	// User is the data of the user to be created.
	User *PostgreSqlUser `json:"user,omitempty"`

	CleanupPolicy *PostgreSqlCleanupPolicy `json:"cleanupPolicy,omitempty"`
}

type PostgreSqlAuth struct {
	// A basic auth username used to authenticate against the PostgreSql instance.
	Username string `json:"username"`
	// A basic auth password used to authenticate against the PostgreSql instance.
	Password esmeta.SecretKeySelector `json:"password"`
}

type PostgreSqlUserAttributesEnum string

const (
	PostgreSqlUserSuperUser       PostgreSqlUserAttributesEnum = "SUPERUSER"
	PostgreSqlUserCreateDb        PostgreSqlUserAttributesEnum = "CREATEDB"
	PostgreSqlUserCreateRole      PostgreSqlUserAttributesEnum = "CREATEROLE"
	PostgreSqlUserReplication     PostgreSqlUserAttributesEnum = "REPLICATION"
	PostgreSqlUserNoInherit       PostgreSqlUserAttributesEnum = "NOINHERIT"
	PostgreSqlUserByPassRls       PostgreSqlUserAttributesEnum = "BYPASSRLS"
	PostgreSqlUserConnectionLimit PostgreSqlUserAttributesEnum = "CONNECTION LIMIT"
	PostgreSqlUserLogin           PostgreSqlUserAttributesEnum = "LOGIN"
	PostgreSqlUserPassword        PostgreSqlUserAttributesEnum = "PASSWORD"
)

type PostgreSqlUser struct {
	// The username of the user to be created.
	Username string `json:"username"`
	// SuffixSize define the size of the random suffix added after the defined username.
	// If not specified, a random suffix of size 8 will be used.
	// If set to 0, no suffix will be added.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=8
	SuffixSize *int `json:"suffixSize,omitempty"`
	// Attributes is the list of PostgreSQL role attributes assigned to this user.
	Attributes []PostgreSqlUserAttribute `json:"attributes,omitempty"`
	// Roles is the list of existing roles that will be granted to this user.
	// If a role does not exist, it will be created without any attributes.
	Roles []string `json:"roles,omitempty"`
	// If set to true, the generator will drop all objects owned by the user
	// before deleting the user during cleanup.
	// If false (default), ownership of all objects will be reassigned
	// to the role specified in `spec.user.reassignTo`.
	// +kubebuilder:default=false
	DestructiveCleanup bool `json:"destructiveCleanup,omitempty"`
	// The name of the role to which all owned objects should be reassigned
	// during cleanup (if DestructiveCleanup is false).
	// If not specified, the role from `spec.auth.username` will be used.
	// If the role does not exist, it will be created with no attributes or roles..
	ReassignTo *string `json:"reassignTo,omitempty"`
}

type PostgreSqlUserAttribute struct {
	// Attribute is the name of the PostgreSQL role attribute to be set for the user.
	// Valid values: SUPERUSER, CREATEDB, CREATEROLE, REPLICATION, NOINHERIT, BYPASSRLS, CONNECTION_LIMIT.
	// +kubebuilder:validation:Enum=SUPERUSER;CREATEDB;CREATEROLE;REPLICATION;NOINHERIT;BYPASSRLS;CONNECTION_LIMIT
	Name string `json:"name"`
	// Optional value for the attribute (e.g., connection limit)
	Value *string `json:"value,omitempty"`
}

type PostgreSqlUserState struct {
	Username string `json:"username,omitempty"`
}

// PostgreSql generates a random postgreSQL based on the
// configuration parameters in spec.
// You can specify the length, characterset and other attributes.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-generators}
type PostgreSql struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSqlSpec              `json:"spec,omitempty"`
	Status genv1alpha1.GeneratorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgreSqlList contains a list of ExternalSecret resources.
type PostgreSqlList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSql `json:"items"`
}
