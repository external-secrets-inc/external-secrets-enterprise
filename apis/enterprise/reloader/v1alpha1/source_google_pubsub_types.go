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
