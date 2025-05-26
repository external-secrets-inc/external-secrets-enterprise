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

import "reflect"

var (
	AWSIAMKeysKind        = reflect.TypeOf(AWSIAMKey{}).Name()
	SendgridKind          = reflect.TypeOf(SendgridAuthorizationToken{}).Name()
	RabbitMQGeneratorKind = reflect.TypeOf(RabbitMQ{}).Name()
	BasicAuthKind         = reflect.TypeOf(BasicAuth{}).Name()
	SSHKind               = reflect.TypeOf(SSH{}).Name()
	Neo4jKind             = reflect.TypeOf(Neo4j{}).Name()
	MongoDBKind           = reflect.TypeOf(MongoDB{}).Name()
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

	SchemeBuilder.Register(&AWSIAMKey{}, &AWSIAMKeysList{})
	SchemeBuilder.Register(&SendgridAuthorizationToken{}, &SendgridAuthorizationTokenList{})
	SchemeBuilder.Register(&RabbitMQ{}, &RabbitMQList{})
	SchemeBuilder.Register(&MongoDB{}, &MongoDBList{})
	SchemeBuilder.Register(&BasicAuth{}, &BasicAuthList{})
	SchemeBuilder.Register(&SSH{}, &SSHList{})
	SchemeBuilder.Register(&Neo4j{}, &Neo4jList{})
}
