// Package v1alpha1 contains API Schema definitions for the reloader v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

// MockConfig represents configuration settings for mock notifications.
type MockConfig struct {
	EmitInterval int32 `json:"emitInterval"`
}
