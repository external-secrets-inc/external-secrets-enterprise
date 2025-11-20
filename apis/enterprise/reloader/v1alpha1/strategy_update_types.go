// Package v1alpha1 contains API Schema definitions for the reloader v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

// UpdateStrategy defines the update strategy.
type UpdateStrategy struct {
	Operation UpdateStrategyOperation `json:"operation"`
	// Required if Operation == Patch
	PatchOperationConfig *PatchOperationConfig `json:"patchOperationConfig,omitempty"`
}

// PatchOperationConfig defines the patch operation configuration.
type PatchOperationConfig struct {
	Path     string `json:"path"`
	Template string `json:"template"`
}

// UpdateStrategyOperation defines the operation to perform on the object.
type UpdateStrategyOperation string

const (
	// UpdateStrategyOperationPatchStatus updates the status of the object.
	UpdateStrategyOperationPatchStatus UpdateStrategyOperation = "PatchStatus"
	// UpdateStrategyOperationPatch updates the object.
	UpdateStrategyOperationPatch UpdateStrategyOperation = "Patch"
	// UpdateStrategyOperationDelete deletes the object.
	UpdateStrategyOperationDelete UpdateStrategyOperation = "Delete"
)
