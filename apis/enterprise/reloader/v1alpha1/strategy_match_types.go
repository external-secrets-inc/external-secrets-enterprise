// Package v1alpha1 contains API Schema definitions for the reloader v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

// MatchStrategy defines the match strategy.
type MatchStrategy struct {
	Path       string      `json:"path"`
	Conditions []Condition `json:"conditions"`
}

// Condition defines a condition to match against.
type Condition struct {
	Value     string             `json:"value"`
	Operation ConditionOperation `json:"operation"`
}

// ConditionOperation defines the operation to perform on the object.
type ConditionOperation string

const (
	// ConditionOperationEqual defines the operation to perform on the object.
	ConditionOperationEqual ConditionOperation = "Equal"
	// ConditionOperationNotEqual defines the operation to perform on the object.
	ConditionOperationNotEqual ConditionOperation = "NotEqual"
	// ConditionOperationContains defines the operation to perform on the object.
	ConditionOperationContains ConditionOperation = "Contains"
	// ConditionOperationNotContains defines the operation to perform on the object.
	ConditionOperationNotContains ConditionOperation = "NotContains"
	// ConditionOperationIn defines the operation to perform on the object.
	ConditionOperationIn ConditionOperation = "RegularExpression"
)
