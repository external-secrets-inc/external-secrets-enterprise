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
	Scan(ctx context.Context, regexes []string, threshold int) ([]SecretInStoreRef, error)
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
