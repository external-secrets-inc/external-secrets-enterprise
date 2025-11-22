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

// AWSSQSConfig contains configuration for AWS SDK.
type AWSSQSConfig struct {
	// QueueURL is the URL of the AWS SDK queue.
	// +required
	QueueURL string `json:"queueURL"`

	// Authentication methods for AWS.
	// +required
	Auth AWSSDKAuth `json:"auth"`

	// MaxNumberOfMessages specifies the maximum number of messages to retrieve from the SDK queue in a single request.
	// +optional
	// +kubebuilder:default=10
	MaxNumberOfMessages int32 `json:"numberOfMessages"`

	// WaitTimeSeconds specifies the duration (in seconds) to wait for messages in the SDK queue before returning.
	// +optional
	// +kubebuilder:default=20
	WaitTimeSeconds int32 `json:"waitTimeSeconds"`

	// VisibilityTimeout specifies the duration (in seconds) that a message received from the SDK queue is hidden from subsequent retrievals.
	// +optional
	// +kubebuilder:default=30
	VisibilityTimeout int32 `json:"visibilityTimeout"`
}

// AWSSDKAuth contains authentication methods for AWS SDK.
type AWSSDKAuth struct {
	AuthMethod string `json:"authMethod"`

	Region string `json:"region"`

	ServiceAccount *ServiceAccountSelector `json:"serviceAccountRef,omitempty"`

	SecretRef *AWSSDKSecretRef `json:"secretRef,omitempty"`
}

// AWSSDKSecretRef contains the AWS SDK credentials.
type AWSSDKSecretRef struct {
	// AccessKeyIDSecretRef is the secret key selector for the access key ID.
	AccessKeyID SecretKeySelector `json:"accessKeyIdSecretRef"`
	// SecretAccessKeySecretRef is the secret key selector for the secret access key.
	SecretAccessKey SecretKeySelector `json:"secretAccessKeySecretRef"`
}
