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
	// +kubebuilder:default=5432
	Port string `json:"port"`
	// Auth contains the credentials or auth configuration
	Auth PostgreSqlAuth `json:"auth"`
	// User is the data of the user to be created.
	User *PostgreSqlUser `json:"user,omitempty"`
}

type PostgreSqlAuth struct {
	// A basic auth username used to authenticate against the PostgreSql instance.
	Username string `json:"username"`
	// A basic auth password used to authenticate against the PostgreSql instance.
	Password SecretKeySelector `json:"password"`
}

type PostgreSqlUserAttributes string

const (
	PostgreSqlUserSuperUser   PostgreSqlUserAttributes = "SUPERUSER"
	PostgreSqlUserCreateDb    PostgreSqlUserAttributes = "CREATEDB"
	PostgreSqlUserCreateRole  PostgreSqlUserAttributes = "CREATEROLE"
	PostgreSqlUserReplication PostgreSqlUserAttributes = "REPLICATION"
)

type PostgreSqlUser struct {
	// The username of the user to be created.
	Username string `json:"username"`
	// SuffixSize define the size of the random suffix added after the defined username.
	// If not specified, a random suffix of size 8 will be used.
	// +kubebuilder:default=8
	SuffixSize *int `json:"suffixSize,omitempty"`
	// Attributes is the list of PostgreSQL role attributes assigned to this user.
	// Valid values: SUPERUSER, CREATEDB, CREATEROLE, REPLICATION.
	// +kubebuilder:validation:Enum=SUPERUSER;CREATEDB;CREATEROLE;REPLICATION;
	Attributes []string `json:"attributes,omitempty"`
	// Roles is the list of existing roles that will be granted to this user.
	// If a role does not exist, it will be created without any attributes.
	Roles []string `json:"roles,omitempty"`
	// If set to true, case the generator needs to clean up the user
	// it will drop everything it owns before dropping him.
	// If not set, all things owned by the user will be reassinged
	// to the user specified in `spec.auth.username`.
	// +kubebuilder:default=false
	DestructiveCleanup bool `json:"destructiveCleanup,omitempty"`
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

	Spec PostgreSqlSpec `json:"spec,omitempty"`
}

// +kubebuilder:object:root=true

// PostgreSqlList contains a list of ExternalSecret resources.
type PostgreSqlList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSql `json:"items"`
}
