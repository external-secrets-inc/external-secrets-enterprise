// Package v1alpha1 contains API Schema definitions for the reloader v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

// GooglePubSubConfig contains configuration for Google Pub/Sub.
type GooglePubSubConfig struct {
	// SubscriptionID is the ID of the Pub/Sub subscription.
	// +required
	SubscriptionID string `json:"subscriptionID"`

	// ProjectID is the GCP project ID where the subscription exists.
	// +required
	ProjectID string `json:"projectID"`

	// Authentication methods for Google Pub/Sub.
	// +optional
	Auth *GooglePubSubAuth `json:"auth,omitempty"`
}

// GooglePubSubAuth contains authentication methods for Google Pub/Sub.
type GooglePubSubAuth struct {
	// +optional
	SecretRef *GCPSMAuthSecretRef `json:"secretRef,omitempty"`
	// +optional
	WorkloadIdentity *GCPWorkloadIdentity `json:"workloadIdentity,omitempty"`
}

// GCPSMAuthSecretRef contains authentication methods for Google Pub/Sub using Secret Access Key.
type GCPSMAuthSecretRef struct {
	// The SecretAccessKey is used for authentication
	// +optional
	SecretAccessKey SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`
}

// GCPWorkloadIdentity contains authentication methods for Google Pub/Sub using Workload Identity.
type GCPWorkloadIdentity struct {
	ServiceAccountRef ServiceAccountSelector `json:"serviceAccountRef"`
	ClusterLocation   string                 `json:"clusterLocation"`
	ClusterName       string                 `json:"clusterName"`
	ClusterProjectID  string                 `json:"clusterProjectID,omitempty"`
}
