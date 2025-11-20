// Package v1alpha1 contains API Schema definitions for the reloader v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

// HashicorpVaultConfig contains configuration for HashicorpVault notifications.
type HashicorpVaultConfig struct {
	// Host is the hostname or IP address to listen on.
	// +required
	Host string `json:"host"`

	// Port is the port number to listen on.
	// +required
	// +kubebuilder:default=8000
	Port int32 `json:"port"`
}
