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

// WorkflowTemplateSpec defines the desired state of WorkflowTemplate.
type WorkflowTemplateSpec struct {
	// Version of the workflow template
	// +required
	Version string `json:"version"`

	// Name is a human-readable name for the workflow template
	// +required
	Name string `json:"name"`

	// Parameters that can be overridden when creating a workflow
	// +optional
	Parameters []TemplateParameter `json:"parameters,omitempty"`

	// Jobs is a map of job names to job definitions
	// +required
	Jobs map[string]Job `json:"jobs"`
}

// TemplateParameter defines a parameter that can be overridden when creating a workflow.
type TemplateParameter struct {
	// Name of the parameter
	// +required
	Name string `json:"name"`

	// Description is a human-readable description of the parameter
	// +optional
	Description string `json:"description,omitempty"`

	// Required indicates whether the parameter must be provided
	// +optional
	Required bool `json:"required,omitempty"`

	// Default value to use if not provided
	// +optional
	Default string `json:"default,omitempty"`
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
