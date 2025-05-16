// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

package jobs

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/common"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/steps"
)

// StepExecutor abstracts the execution of a workflow step.
type StepExecutor interface {
	Execute(ctx context.Context, client client.Client, wf *workflows.Workflow, data map[string]interface{}) (map[string]interface{}, error)
}

// createExecutor returns a StepExecutor based on the step type.
func createExecutor(step workflows.Step, c client.Client, scheme runtime.Scheme, logger logr.Logger, manager secretstore.ManagerInterface) (StepExecutor, error) {
	switch {
	case step.Pull != nil:
		return steps.NewPullStepExecutor(step.Pull, c, manager), nil
	case step.Push != nil:
		return steps.NewPushStepExecutor(step.Push, c, manager), nil
	case step.Debug != nil:
		return steps.NewDebugStepExecutor(step.Debug), nil
	case step.Transform != nil:
		return steps.NewTransformStepExecutor(step.Transform), nil
	case step.Generator != nil:
		return steps.NewGeneratorStepExecutor(step.Generator, c, &scheme, manager), nil
	case step.JavaScript != nil:
		return steps.NewJavaScriptExecutor(step.JavaScript, logger), nil
	default:
		return nil, fmt.Errorf("unknown step type")
	}
}

// StepContext holds the context for executing a workflow step.
type StepContext struct {
	Client    client.Client
	Workflow  *workflows.Workflow
	JobStatus *workflows.JobStatus
	Scheme    *runtime.Scheme
	Logger    logr.Logger
	Data      map[string]interface{}
	Manager   secretstore.ManagerInterface
}

// InitializeStepStatus initializes or retrieves the status for a step.
func InitializeStepStatus(jobStatus *workflows.JobStatus, stepKey string) workflows.StepStatus {
	stepStatus := jobStatus.StepStatuses[stepKey]

	// Skip if the step has already been processed
	if stepStatus.Phase == workflows.StepPhaseSucceeded || stepStatus.Phase == workflows.StepPhaseFailed {
		return stepStatus
	}

	// Initialize if not already running
	if stepStatus.Phase == "" {
		stepStatus.Phase = workflows.StepPhaseRunning
		now := metav1.Now()
		stepStatus.StartTime = &now
		jobStatus.StepStatuses[stepKey] = stepStatus
	}

	return stepStatus
}

// ExecuteStep executes a workflow step and updates its status.
func ExecuteStep(
	ctx context.Context,
	stepCtx StepContext,
	step workflows.Step,
	stepKey string,
) error {
	// Initialize step status
	stepStatus := InitializeStepStatus(stepCtx.JobStatus, stepKey)

	// Create executor for the step
	stepExecutor, err := createExecutor(step, stepCtx.Client, *stepCtx.Scheme, stepCtx.Logger, stepCtx.Manager)
	if err != nil {
		return markStepFailed(stepCtx.JobStatus, stepKey, stepStatus, err)
	}

	// Execute the step
	outputs, err := stepExecutor.Execute(ctx, stepCtx.Client, stepCtx.Workflow, stepCtx.Data)
	if err != nil {
		return markStepFailed(stepCtx.JobStatus, stepKey, stepStatus, err)
	}

	// Process and store outputs
	serializedOutputs, err := SerializeStepOutputs(outputs, step)
	if err != nil {
		return fmt.Errorf("failed to serialize step outputs: %w", err)
	}

	// Mark step as succeeded
	stepStatus.Phase = workflows.StepPhaseSucceeded
	now := metav1.Now()
	stepStatus.CompletionTime = &now
	stepStatus.Outputs = serializedOutputs
	stepCtx.JobStatus.StepStatuses[stepKey] = stepStatus

	return nil
}

// markStepFailed marks a step as failed and returns an error.
func markStepFailed(jobStatus *workflows.JobStatus, stepKey string, stepStatus workflows.StepStatus, err error) error {
	stepStatus.Phase = workflows.StepPhaseFailed
	stepStatus.Message = err.Error()
	jobStatus.StepStatuses[stepKey] = stepStatus
	return fmt.Errorf("step %s failed: %w", stepKey, err)
}

// SerializeStepOutputs converts step outputs to a serializable format for storage.
// It delegates to common.ProcessOutputs for the actual serialization and masking.
func SerializeStepOutputs(outputs map[string]interface{}, step workflows.Step) (map[string]string, error) {
	if outputs == nil {
		outputs = make(map[string]interface{})
	}

	// Use the common ProcessOutputs function to handle serialization and masking
	// Ignore the sensitive values return value as we only need the serialized outputs
	serialized, _, err := common.ProcessOutputs(outputs, step)
	return serialized, err
}

// MarkJobCompleted marks a job as completed successfully.
func MarkJobCompleted(jobStatus *workflows.JobStatus) {
	jobStatus.Phase = workflows.JobPhaseSucceeded
	now := metav1.Now()
	jobStatus.CompletionTime = &now
}

// flattenJobStatuses returns a nested map of job and step outputs.
func flattenJobStatuses(jobStatuses map[string]workflows.JobStatus) map[string]map[string]map[string]string {
	flattened := make(map[string]map[string]map[string]string)
	for jobName, jobStatus := range jobStatuses {
		stepsMap := make(map[string]map[string]string)
		for stepName, stepStatus := range jobStatus.StepStatuses {
			stepsMap[stepName] = stepStatus.Outputs
		}
		flattened[jobName] = stepsMap
	}
	return flattened
}

// BuildWorkflowContext builds the common data context for workflow steps.
func BuildWorkflowContext(wf *workflows.Workflow) map[string]interface{} {
	return map[string]interface{}{
		"global": map[string]interface{}{
			"variables": wf.Spec.Variables,
			"jobs":      flattenJobStatuses(wf.Status.JobStatuses),
		},
	}
}

// JobExecutionContext holds the common context for job execution.
type JobExecutionContext struct {
	Client    client.Client
	Workflow  *workflows.Workflow
	JobName   string
	JobStatus *workflows.JobStatus
	Scheme    *runtime.Scheme
	Logger    logr.Logger
	Data      map[string]interface{}
	Manager   secretstore.ManagerInterface
}

// NewJobExecutionContext creates a new job execution context with workflow data.
func NewJobExecutionContext(
	client client.Client,
	wf *workflows.Workflow,
	jobName string,
	jobStatus *workflows.JobStatus,
	scheme *runtime.Scheme,
	logger logr.Logger,
	manager secretstore.ManagerInterface,
) *JobExecutionContext {
	return &JobExecutionContext{
		Client:    client,
		Workflow:  wf,
		JobName:   jobName,
		JobStatus: jobStatus,
		Scheme:    scheme,
		Logger:    logger,
		Data:      BuildWorkflowContext(wf),
		Manager:   manager,
	}
}

// ExecuteStepWithContext executes a single step with the provided context.
func ExecuteStepWithContext(
	ctx context.Context,
	jobCtx *JobExecutionContext,
	step workflows.Step,
	stepKey string,
) error {
	// Create step context from job context
	stepCtx := StepContext{
		Client:    jobCtx.Client,
		Workflow:  jobCtx.Workflow,
		JobStatus: jobCtx.JobStatus,
		Scheme:    jobCtx.Scheme,
		Logger:    jobCtx.Logger,
		Data:      jobCtx.Data,
		Manager:   jobCtx.Manager,
	}

	// Execute the step using the existing ExecuteStep function
	err := ExecuteStep(ctx, stepCtx, step, stepKey)
	if err != nil {
		return err
	}

	// Update the job context's Data map with the latest workflow context
	// This ensures that subsequent steps can access outputs from previous steps
	jobCtx.Data = BuildWorkflowContext(jobCtx.Workflow)

	return nil
}

// CompleteJob marks a job as completed and returns nil.
func CompleteJob(jobStatus *workflows.JobStatus) error {
	MarkJobCompleted(jobStatus)
	return nil
}
