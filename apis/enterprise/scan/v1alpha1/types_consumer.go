// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConsumerConditionType string

const (
	ConsumerLatestVersion ConsumerConditionType = "UsingLatestVersion"
)

const (
	ConsumerLocationsUpToDate  = "LocationsUpToDate"
	ConsumerLocationsOutOfDate = "LocationsOutOfDate"
	ConsumerPodsNotReady       = "PodsNotReady"
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
	VMProcess   *VMProcessSpec   `json:"vmProcess,omitempty"`
	GitHubActor *GitHubActorSpec `json:"githubActor,omitempty"`
	K8sWorkload *K8sWorkloadSpec `json:"k8sWorkload,omitempty"`
}

type ConsumerStatus struct {
	ObservedIndex map[string]tgtv1alpha1.SecretUpdateRecord `json:"observedIndex,omitempty"`
	Locations     []tgtv1alpha1.SecretInStoreRef            `json:"locations,omitempty"`
	Conditions    []metav1.Condition                        `json:"conditions,omitempty"`
	Pods          []K8sPodItem                              `json:"pods,omitempty"`
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

type K8sPodItem struct {
	Name     string `json:"name"`
	UID      string `json:"uid,omitempty"`
	NodeName string `json:"nodeName,omitempty"`
	Phase    string `json:"phase,omitempty"`
	Ready    bool   `json:"ready"`
	Reason   string `json:"reason,omitempty"`
}

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
