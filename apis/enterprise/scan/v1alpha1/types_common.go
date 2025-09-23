// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type SecretInStoreRef struct {
	Name       string    `json:"name"`
	Kind       string    `json:"kind"`
	APIVersion string    `json:"apiVersion"`
	RemoteRef  RemoteRef `json:"remoteRef"`
}

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
