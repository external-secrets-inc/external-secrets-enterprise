// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type JobSpec struct {
	// Constrains this job to a given set of SecretStores / Targets.
	// By default it will run against all SecretStores / Targets on the Job namespace.
	Constraints *JobConstraints `json:"constraints,omitempty"`
	// Defines the RunPolicy for this job (Poll/OnChange/Once)
	// +kubebuilder:validation:Enum=Poll;OnChange;Once
	RunPolicy JobRunPolicy `json:"runPolicy,omitempty"`
	// Defines the interval for this job if Policy is Poll(Poll/OnChange/Once)
	Interval metav1.Duration `json:"interval,omitempty"`
	// TODO - also implement Cron Schedulingf
	// Define the interval to wait before forcing reconcile if job froze at running state
	// +kubebuilder:default="10m"
	JobTimeout metav1.Duration `json:"jobTimeout,omitempty"`
}

type JobRunPolicy string

const (
	JobRunPolicyPull     JobRunPolicy = "Poll"
	JobRunPolicyOnChange JobRunPolicy = "OnChange"
	JobRunPolicyOnce     JobRunPolicy = "Once"
)

type JobConstraints struct {
	SecretStoreConstraints []SecretStoreConstraint `json:"secretStoreConstraints,omitempty"`
	TargetConstraints      []TargetConstraint      `json:"targetConstraints,omitempty"`
}

type SecretStoreConstraint struct {
	MatchExpressions []metav1.LabelSelector `json:"matchExpression,omitempty"`
	MatchLabels      map[string]string      `json:"matchLabels,omitempty"`
}

type TargetConstraint struct {
	Kind             string                 `json:"kind,omitempty"`
	APIVersion       string                 `json:"apiVersion,omitempty"`
	MatchExpressions []metav1.LabelSelector `json:"matchExpression,omitempty"`
	MatchLabels      map[string]string      `json:"matchLabels,omitempty"`
}

type JobStatus struct {
	// ObservedSecretStoresDigest is a digest of the SecretStores that were used in the last run.
	// +optional
	ObservedSecretStoresDigest string `json:"observedSecretStoresDigest,omitempty"`
	// ObservedTargetsDigest is a digest of the Targets that were used in the last run.
	// +optional
	ObservedTargetsDigest string `json:"observedTargetsDigest,omitempty"`

	LastRunTime metav1.Time        `json:"lastRunTime,omitempty"`
	RunStatus   JobRunStatus       `json:"runStatus,omitempty"`
	Conditions  []metav1.Condition `json:"conditions,omitempty"`
}

type JobRunStatus string

const (
	JobRunStatusRunning   JobRunStatus = "Running"
	JobRunStatusSucceeded JobRunStatus = "Succeeded"
	JobRunStatusFailed    JobRunStatus = "Failed"
)

// Job is the schema to run a scan job over targets
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-scan}
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              JobSpec   `json:"spec,omitempty"`
	Status            JobStatus `json:"status,omitempty"`
}

// JobList contains a list of Job resources.
// +kubebuilder:object:root=true
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}
