// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

package jobs

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"
)

// SwitchJobExecutor handles execution of switch jobs.
type SwitchJobExecutor struct {
	job     *workflows.SwitchJob
	log     logr.Logger
	scheme  *runtime.Scheme
	manager secretstore.ManagerInterface
}

// NewSwitchJobExecutor creates a new SwitchJobExecutor.
func NewSwitchJobExecutor(job *workflows.SwitchJob, scheme *runtime.Scheme, log logr.Logger, manager secretstore.ManagerInterface) *SwitchJobExecutor {
	return &SwitchJobExecutor{
		job:     job,
		scheme:  scheme,
		log:     log,
		manager: manager,
	}
}

// Execute processes a switch job by evaluating each case's condition and executing
// the steps of the first case whose condition evaluates to true.
func (e *SwitchJobExecutor) Execute(ctx context.Context, client client.Client, wf *workflows.Workflow, jobName string, jobStatus *workflows.JobStatus) error {
	e.log.Info("Executing switch job", "job", jobName)

	if e.job == nil || len(e.job.Cases) == 0 {
		return fmt.Errorf("switch job has no cases defined")
	}

	// Create job execution context
	jobCtx := NewJobExecutionContext(client, wf, jobName, jobStatus, e.scheme, e.log, e.manager)

	// Evaluate each case's condition in order
	for i, switchCase := range e.job.Cases {
		// Resolve the condition template
		resolvedCondition, err := templates.ResolveTemplate(switchCase.Condition, jobCtx.Data)
		if err != nil {
			return fmt.Errorf("failed to resolve condition template for case %d: %w", i, err)
		}

		// Convert the resolved condition to a boolean
		conditionValue, err := strconv.ParseBool(resolvedCondition)
		if err != nil {
			return fmt.Errorf("condition for case %d did not resolve to a boolean value: %w", i, err)
		}

		// If the condition is true, execute this case's steps
		if conditionValue {
			e.log.Info("Condition evaluated to true, executing case", "job", jobName, "case", i)

			// Process each step sequentially
			for _, step := range switchCase.Steps {
				if err := ExecuteStepWithContext(ctx, jobCtx, step, step.Name); err != nil {
					return err
				}
			}

			// This case was executed, mark the job as succeeded
			return CompleteJob(jobStatus)
		}
	}

	// If we get here, no case condition evaluated to true
	e.log.Info("No case conditions evaluated to true", "job", jobName)

	// We still consider the job successful even if no case was executed
	return CompleteJob(jobStatus)
}
