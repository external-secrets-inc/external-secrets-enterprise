// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

// Package steps provides workflow step executors.
package steps

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/enterprise/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/controllers/workflow/templates"
)

// DebugStepExecutor executes debug steps.
type DebugStepExecutor struct {
	Step *esapi.DebugStep
}

// NewDebugStepExecutor creates a new debug step executor.
func NewDebugStepExecutor(step *esapi.DebugStep) *DebugStepExecutor {
	return &DebugStepExecutor{
		Step: step,
	}
}

// Execute executes the debug step.
func (e *DebugStepExecutor) Execute(_ context.Context, _ client.Client, _ *esapi.Workflow, data map[string]interface{}, _ string) (map[string]interface{}, error) {
	message, err := templates.ResolveTemplate(e.Step.Message, data)
	if err != nil {
		return nil, fmt.Errorf("resolving message: %w", err)
	}
	fmt.Println("Debug message:", message)

	// Return the message as an output
	return map[string]interface{}{
		"message": message,
	}, nil
}
