// Copyright External Secrets Inc. 2025
// All rights reserved

package v1alpha1

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil
type TargetProvider interface {
	NewClient(ctx context.Context, client client.Client, target client.Object) (ScanTarget, error)
}

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil
type ScanTarget interface {
	ScanForSecrets(ctx context.Context, regexes []string, threshold int) ([]SecretInStoreRef, error)
	ScanForConsumers(ctx context.Context, location SecretInStoreRef) ([]ConsumerFinding, error)
}

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

type ConsumerFinding struct {
	Location    SecretInStoreRef  `json:"location"`
	Kind        string            `json:"kind"`
	ID          string            `json:"externalID"`
	DisplayName string            `json:"displayName,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}
