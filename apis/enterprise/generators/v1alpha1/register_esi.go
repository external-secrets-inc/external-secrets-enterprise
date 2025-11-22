// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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
	"reflect"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "generators.external-secrets.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	// AddToScheme is used to add go types to the GroupVersionKind scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

var (
	// AWSIAMKeysKind is the type name of the AWS IAM keys generator.
	AWSIAMKeysKind = reflect.TypeOf(AWSIAMKey{}).Name()
	// SendgridKind is the type name of the Sendgrid generator.
	SendgridKind = reflect.TypeOf(SendgridAuthorizationToken{}).Name()
	// RabbitMQGeneratorKind is the type name of the RabbitMQ generator.
	RabbitMQGeneratorKind = reflect.TypeOf(RabbitMQ{}).Name()
	// BasicAuthKind is the type name of the Basic Auth generator.
	BasicAuthKind = reflect.TypeOf(BasicAuth{}).Name()
	// FederationKind is the type name of the Federation generator.
	FederationKind = reflect.TypeOf(Federation{}).Name()
	// SSHKind is the type name of the SSH generator.
	SSHKind = reflect.TypeOf(SSH{}).Name()
	// Neo4jKind is the type name of the Neo4j generator.
	Neo4jKind = reflect.TypeOf(Neo4j{}).Name()
	// MongoDBKind is the type name of the MongoDB generator.
	MongoDBKind = reflect.TypeOf(MongoDB{}).Name()
	// PostgreSQLKind is the type name of the PostgreSQL generator.
	PostgreSQLKind = reflect.TypeOf(PostgreSQL{}).Name()
	// OpenAIKind is the type name of the OpenAI generator.
	OpenAIKind = reflect.TypeOf(OpenAI{}).Name()
)

func init() {
	/*
		===============================================================================
		 NOTE: when adding support for new kinds of generators:
		  1. register the struct types in `SchemeBuilder` (right below this note)
		  2. update the `kubebuilder:validation:Enum` annotation for GeneratorRef.Kind (apis/externalsecrets/v1beta1/externalsecret_types.go)
		  3. add it to the imports of (pkg/generator/register/register.go)
		  4. add it to the ClusterRole called "*-controller" (deploy/charts/external-secrets/templates/rbac.yaml)
		  5. support it in ClusterGenerator:
			  - add a new GeneratorKind enum value (apis/generators/v1alpha1/types_cluster.go)
			  - update the `kubebuilder:validation:Enum` annotation for the GeneratorKind enum
			  - add a spec field to GeneratorSpec (apis/generators/v1alpha1/types_cluster.go)
			  - update the clusterGeneratorToVirtual() function (pkg/utils/resolvers/generator.go)
		===============================================================================
	*/

	genv1alpha1.SchemeBuilder.Register(&AWSIAMKey{}, &AWSIAMKeyList{})
	genv1alpha1.SchemeBuilder.Register(&SendgridAuthorizationToken{}, &SendgridAuthorizationTokenList{})
	genv1alpha1.SchemeBuilder.Register(&RabbitMQ{}, &RabbitMQList{})
	genv1alpha1.SchemeBuilder.Register(&MongoDB{}, &MongoDBList{})
	genv1alpha1.SchemeBuilder.Register(&BasicAuth{}, &BasicAuthList{})
	genv1alpha1.SchemeBuilder.Register(&SSH{}, &SSHList{})
	genv1alpha1.SchemeBuilder.Register(&Neo4j{}, &Neo4jList{})
	genv1alpha1.SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
	genv1alpha1.SchemeBuilder.Register(&OpenAI{}, &OpenAIList{})
	genv1alpha1.SchemeBuilder.Register(&Federation{}, &FederationList{})

	// Only used so our generic_generator is able to create the appropriate methods
	SchemeBuilder.Register(&AWSIAMKey{}, &AWSIAMKeyList{})
	SchemeBuilder.Register(&SendgridAuthorizationToken{}, &SendgridAuthorizationTokenList{})
	SchemeBuilder.Register(&RabbitMQ{}, &RabbitMQList{})
	SchemeBuilder.Register(&MongoDB{}, &MongoDBList{})
	SchemeBuilder.Register(&BasicAuth{}, &BasicAuthList{})
	SchemeBuilder.Register(&SSH{}, &SSHList{})
	SchemeBuilder.Register(&Neo4j{}, &Neo4jList{})
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
	SchemeBuilder.Register(&OpenAI{}, &OpenAIList{})
	SchemeBuilder.Register(&Federation{}, &FederationList{})
}
