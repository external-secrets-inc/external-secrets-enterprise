// Package v1alpha1 contains API Schema definitions for the scan v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// SecretInStoreRef defines a reference to a secret in a secret store.
type SecretInStoreRef struct {
	Name       string    `json:"name"`
	Kind       string    `json:"kind"`
	APIVersion string    `json:"apiVersion"`
	RemoteRef  RemoteRef `json:"remoteRef"`
}

// RemoteRef defines a reference to a remote secret.
type RemoteRef struct {
	Key        string `json:"key"`
	Property   string `json:"property,omitempty"`
	StartIndex *int   `json:"startIndex,omitempty"`
	EndIndex   *int   `json:"endIndex,omitempty"`
}

// SecretUpdateRecord defines the timestamp when a PushSecret was applied to a secret.
type SecretUpdateRecord struct {
	Timestamp  metav1.Time `json:"timestamp"`
	SecretHash string      `json:"secretHash"`
}
