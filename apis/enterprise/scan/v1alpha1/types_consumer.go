// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConsumerConditionType string

const (
	ConsumerLatestVersion ConsumerConditionType = "UsingLatestVersion"
)

const (
	ConsumerLocationsUpToDate  = "LocationsUpToDate"
	ConsumerLocationsOutOfDate = "LocationsOutOfDate"
	ConsumerWorkloadReady      = "WorkloadReady"
	ConsumerWorkloadNotReady   = "WorkloadNotReady"
	ConsumerNotReady           = "ConsumerNotReady"
)

type ConsumerSpec struct {
	Target TargetReference `json:"target"`

	// Type discriminates which payload below is populated.
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// A stable ID for correlation across scans.
	// +kubebuilder:validation:MinLength=1
	ID string `json:"id"`

	// Human readable name for UIs.
	DisplayName string `json:"displayName"`

	// Exactly one of the following should be set according to Type.
	// +kubebuilder:validation:Required
	Attributes ConsumerAttrs `json:"attributes"`
}

type ConsumerAttrs struct {
	VMProcess   *VMProcessSpec   `json:"vmProcess,omitempty"`
	GitHubActor *GitHubActorSpec `json:"gitHubActor,omitempty"`
	K8sWorkload *K8sWorkloadSpec `json:"k8sWorkload,omitempty"`
}

type ConsumerStatus struct {
	ObservedIndex map[string]SecretUpdateRecord `json:"observedIndex,omitempty"`
	Locations     []SecretInStoreRef            `json:"locations,omitempty"`
	Conditions    []metav1.Condition            `json:"conditions,omitempty"`
}

type TargetReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// VMProcessSpec describes a process on a VM/host.
type VMProcessSpec struct {
	Hostname   string   `json:"hostname"`             // e.g., "ip-10-0-1-23"
	PID        int64    `json:"pid"`                  // process id
	Executable string   `json:"executable,omitempty"` // "/usr/sbin/nginx"
	Cmdline    []string `json:"cmdline,omitempty"`    // ["nginx","-g","daemon off;"]
	User       string   `json:"user,omitempty"`       // "www-data"
}

// GitHubActorSpec describes who/what is interacting with a repo.
type GitHubActorSpec struct {
	// Repo slug "owner/name" for context (e.g., "acme/api").
	Repository string `json:"repository"`
	// ActorType: "User" | "App" | "Bot" (GitHub notions)
	// +kubebuilder:validation:Enum=User;App;Bot
	ActorType  string `json:"actorType"`
	ActorLogin string `json:"actorLogin,omitempty"` // "octocat"
	ActorID    string `json:"actorID,omitempty"`    // stable numeric id if known
	// Optional context that led to detection (push/clone/workflow).
	Event string `json:"event,omitempty"` // "clone","workflow","push"
	// Optional: workflow/job id when usage came from Actions
	WorkflowRunID string `json:"workflowRunID,omitempty"`
}

// K8sWorkloadSpec describes the workload that is interacting with a kubernetes target.
type K8sWorkloadSpec struct {
	ClusterName string `json:"clusterName,omitempty"`

	Namespace string `json:"namespace"`

	// Workload identity (top controller or naked Pod as fallback)
	// e.g., Kind="Deployment", Group="apps", Version="v1", Name="api"
	WorkloadKind    string `json:"workloadKind"`
	WorkloadGroup   string `json:"workloadGroup,omitempty"`
	WorkloadVersion string `json:"workloadVersion,omitempty"`
	WorkloadName    string `json:"workloadName"`
	WorkloadUID     string `json:"workloadUID,omitempty"`

	// Convenience string for UIs: "deployment.apps/api"
	Controller string `json:"controller,omitempty"`
}

type ConsumerFinding struct {
	ObservedIndex SecretUpdateRecord `json:"observedIndex"`
	Location      SecretInStoreRef   `json:"location"`
	Type          string             `json:"type"`
	ID            string             `json:"externalID"`
	DisplayName   string             `json:"displayName,omitempty"`
	Attributes    ConsumerAttrs      `json:"attributes"`
}

// Consumer is the schema to store duplicate findings from a job
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-scan}
type Consumer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ConsumerSpec   `json:"spec,omitempty"`
	Status            ConsumerStatus `json:"status,omitempty"`
}

// JobList contains a list of Job resources.
// +kubebuilder:object:root=true
type ConsumerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Consumer `json:"items"`
}
