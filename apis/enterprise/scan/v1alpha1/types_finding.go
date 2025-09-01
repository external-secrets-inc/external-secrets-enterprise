// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FindingSpec struct {
	ID             string                `json:"id"`
	DisplayName    string                `json:"displayName,omitempty"`
	Hash           string                `json:"hash,omitempty"` // Hash of the finding (salted)
	RunTemplateRef *RunTemplateReference `json:"runTemplateRef,omitempty"`
}

type RunTemplateReference struct {
	Name string `json:"name"`
}

type FindingStatus struct {
	Locations []tgtv1alpha1.SecretInStoreRef `json:"locations,omitempty"`
}

// Finding is the schema to store duplicate findings from a job
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-scan}
type Finding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FindingSpec   `json:"spec,omitempty"`
	Status            FindingStatus `json:"status,omitempty"`
}

// JobList contains a list of Job resources.
// +kubebuilder:object:root=true
type FindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Finding `json:"items"`
}
