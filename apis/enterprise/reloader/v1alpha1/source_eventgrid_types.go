// Package v1alpha1 contains API Schema definitions for the reloader v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

// AzureEventGridConfig contains configuration for Azure Event Grid notifications.
type AzureEventGridConfig struct {
	Host string `json:"host"`

	// +required
	// +kubebuilder:default=8080
	Port int32 `json:"port"`

	Subscriptions []string `json:"subscriptions"`
}
