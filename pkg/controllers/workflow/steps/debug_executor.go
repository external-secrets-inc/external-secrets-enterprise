// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package steps

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"
)

type DebugStepExecutor struct {
	Step *esapi.DebugStep
}

func NewDebugStepExecutor(step *esapi.DebugStep) *DebugStepExecutor {
	return &DebugStepExecutor{
		Step: step,
	}
}

func (e *DebugStepExecutor) Execute(ctx context.Context, client client.Client, wf *esapi.Workflow, data map[string]interface{}, jobName string) (map[string]interface{}, error) {
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
