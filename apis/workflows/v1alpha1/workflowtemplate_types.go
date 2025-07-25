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
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowTemplateSpec defines the desired state of WorkflowTemplate.
type WorkflowTemplateSpec struct {
	// Version of the workflow template
	// +required
	Version string `json:"version"`

	// Name is a human-readable name for the workflow template
	// +required
	Name string `json:"name"`

	// ParameterGroups that can be overridden when creating a workflow
	// +optional
	ParameterGroups []ParameterGroup `json:"parameterGroups,omitempty"`

	// Jobs is a map of job names to job definitions
	// +required
	Jobs map[string]Job `json:"jobs"`
}

// ParameterGroup defines a group of parameters with a name and description.
type ParameterGroup struct {
	// Name of the parameter group
	// +required
	Name string `json:"name"`

	// Description is a human-readable description of the parameter group
	// +optional
	Description string `json:"description,omitempty"`

	// Parameters contained in this group
	// +required
	Parameters []Parameter `json:"parameters"`
}

// ParameterType represents the data type of a parameter
// +kubebuilder:validation:Pattern=`^(string|number|bool|object|secret|time|namespace|secretstore|externalsecret|clustersecretstore|k8ssecret|array\[secretstore\]|generator\[[a-zA-Z0-9_-]+\]|array\[generator\[[a-zA-Z0-9_-]+\]\]|secretlocation|array\[secretlocation\]|finding|array\[finding\]|object\[([a-zA-Z0-9_-]+)\]([a-zA-Z0-9_\-\[\]]+))$`
//
//nolint:lll
type ParameterType string

const (
	// ParameterTypeString Primitive types.
	ParameterTypeString       ParameterType = "string"
	ParameterTypeNumber       ParameterType = "number"
	ParameterTypeBool         ParameterType = "bool"
	ParameterTypeObject       ParameterType = "object"
	ParameterTypeSecret       ParameterType = "secret"
	ParameterTypeTime         ParameterType = "time"
	ParameterTypeCustomObject ParameterType = `^object\[([a-zA-Z0-9_-]+)\](namespace|secretstore|externalsecret|clustersecretstore|secretlocation|finding|generator\[[a-zA-Z0-9_-]+\]|array\[(?:secretstore|secretlocation|finding|generator\[[a-zA-Z0-9_-]+\])\])$`

	// ParameterTypeNamespace Kubernetes resource types.
	ParameterTypeNamespace          ParameterType = "namespace"
	ParameterTypeSecretStore        ParameterType = "secretstore"
	ParameterTypeExternalSecret     ParameterType = "externalsecret"
	ParameterTypeClusterSecretStore ParameterType = "clustersecretstore"
	ParameterTypeGenerator          ParameterType = `^generator\[([a-zA-Z0-9_-]+)\]$`
	ParameterTypeSecretLocation     ParameterType = "secretlocation"
	ParameterTypeFinding            ParameterType = "finding"

	// Array Types (add as needed).
	ParameterTypeSecretStoreArray    ParameterType = "array[secretstore]"
	ParameterTypeGeneratorArray      ParameterType = `^array\[generator\[([a-zA-Z0-9_-]+)\]\]$`
	ParameterTypeSecretLocationArray ParameterType = "array[secretlocation]"
	ParameterTypeFindingArray        ParameterType = "array[finding]"
)

// SecretStoreParameter defines a parameter to be passed to a secret store type.
type SecretStoreParameterType struct {
	// Name is the name of the secretstore.
	Name string `json:"name"`
}

// GeneratorParameter defines a parameter to be passed to a generator type.
type GeneratorParameterType struct {
	// Name is the name of the generator.
	Name *string `json:"name,omitempty"`

	// Kind defines the kind of the generator. It can be 'any'
	Kind *string `json:"kind,omitempty"`
}

// SecretLocationParameter defines a parameter to be passed to a secret location type.
type SecretLocationParameterType = tgtv1alpha1.SecretInStoreRef

// FindingParameter defines a parameter to be passed to a secret store type.
type FindingParameterType struct {
	// Name is the name of the secretstore.
	Name string `json:"name"`
}

// ResourceConstraints defines constraints for Kubernetes resource selection.
type ResourceConstraints struct {
	// Namespace restricts resource selection to specific namespace(s)
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// LabelSelector for filtering resources
	// +optional
	LabelSelector map[string]string `json:"labelSelector,omitempty"`

	// AllowCrossNamespace indicates if resources from other namespaces can be selected
	// Only applies to cluster-scoped selections
	// +optional
	AllowCrossNamespace bool `json:"allowCrossNamespace,omitempty"`
}

// ParameterValidation defines validation rules for parameters.
type ParameterValidation struct {
	// MinItems minimum number of items for multi-select (only when AllowMultiple=true)
	// +optional
	MinItems *int `json:"minItems,omitempty"`

	// MaxItems maximum number of items for multi-select (only when AllowMultiple=true)
	// +optional
	MaxItems *int `json:"maxItems,omitempty"`
}

// Parameter defines a parameter that can be overridden when creating a workflow.
type Parameter struct {
	// Name of the parameter
	// +required
	Name string `json:"name"`

	// Description is a human-readable description of the parameter
	// +optional
	Description string `json:"description,omitempty"`

	// Type specifies the data type of the parameter
	// For array/multi-value parameters, use allowMultiple: true with the appropriate type
	// +optional
	Type ParameterType `json:"type,omitempty"`

	// Required indicates whether the parameter must be provided
	// +optional
	Required bool `json:"required,omitempty"`

	// Default value to use if not provided
	// +optional
	Default string `json:"default,omitempty"`

	// AllowMultiple indicates if multiple values can be selected
	// When true, the parameter accepts an array of values of the specified type
	// +optional
	AllowMultiple bool `json:"allowMultiple,omitempty"`

	// ResourceConstraints for Kubernetes resource types
	// +optional
	ResourceConstraints *ResourceConstraints `json:"resourceConstraints,omitempty"`

	// Validation constraints
	// MinItems and MaxItems apply when allowMultiple is true
	// +optional
	Validation *ParameterValidation `json:"validation,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// WorkflowTemplate is the Schema for the workflowtemplates API.
type WorkflowTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec WorkflowTemplateSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// WorkflowTemplateList contains a list of WorkflowTemplate.
type WorkflowTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkflowTemplate{}, &WorkflowTemplateList{})
}
