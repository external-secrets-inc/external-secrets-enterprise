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
