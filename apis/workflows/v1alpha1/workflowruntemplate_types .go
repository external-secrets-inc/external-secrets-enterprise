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

// WorkflowRunTemplateSpec defines the desired state of WorkflowRunTemplate.
type WorkflowRunTemplateSpec struct {
	RunSpec WorkflowRunSpec `json:"runSpec"`
	// +kubebuilder:default={once:{}}
	RunPolicy RunPolicy `json:"runPolicy"`
	//+kubebuilder:default=3
	//+kubebuilder:validation:Minimum=1
	RevisionHistoryLimit int `json:"revisionHistoryLimit,omitempty"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type RunPolicy struct {
	Once      *RunPolicyOnce      `json:"once,omitempty"`
	Scheduled *RunPolicyScheduled `json:"scheduled,omitempty"`
	OnChange  *RunPolicyOnChange  `json:"onChange,omitempty"`
}

type RunPolicyOnce struct {
}

type RunPolicyOnChange struct {
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type RunPolicyScheduled struct {
	Every *metav1.Duration `json:"every,omitempty"`
	Cron  *string          `json:"cron,omitempty"`
}

// WorkflowRunStatus defines the observed state of WorkflowRun.
type WorkflowRunTemplateStatus struct {
	//+optional
	LastRunTime *metav1.Time `json:"lastRunTime,omitempty"`
	//+optional
	RunStatuses []NamedWorkflowRunStatus `json:"runs,omitempty"`
	// Conditions represent the latest available observations of the WorkflowRun's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	SyncedResourceVersion string `json:"syncedResourceVersion,omitempty"`
}

type NamedWorkflowRunStatus struct {
	RunName           string `json:"runName"`
	WorkflowRunStatus `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="TEMPLATE",type="string",JSONPath=".spec.runSpec.templateRef.name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// WorkflowRunTemplate is the Schema for the workflowruntemplates API.
type WorkflowRunTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowRunTemplateSpec   `json:"spec"`
	Status WorkflowRunTemplateStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WorkflowRunTemplateList contains a list of WorkflowRun.
type WorkflowRunTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkflowRunTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WorkflowRunTemplate{}, &WorkflowRunTemplateList{})
}
