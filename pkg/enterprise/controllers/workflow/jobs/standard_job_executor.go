// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

package jobs

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/enterprise/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
)

// JobExecutor abstracts the execution of a workflow job.
type JobExecutor interface {
	Execute(ctx context.Context, client client.Client, wf *workflows.Workflow, jobName string, jobStatus *workflows.JobStatus) error
}

// StandardJobExecutor handles execution of standard jobs.
type StandardJobExecutor struct {
	job     *workflows.StandardJob
	log     logr.Logger
	scheme  *runtime.Scheme
	manager secretstore.ManagerInterface
}

// NewStandardJobExecutor creates a new StandardJobExecutor.
func NewStandardJobExecutor(job *workflows.StandardJob, scheme *runtime.Scheme, log logr.Logger, manager secretstore.ManagerInterface) *StandardJobExecutor {
	return &StandardJobExecutor{
		job:     job,
		scheme:  scheme,
		log:     log,
		manager: manager,
	}
}

// Execute processes all steps within a standard job.
func (e *StandardJobExecutor) Execute(ctx context.Context, client client.Client, wf *workflows.Workflow, jobName string, jobStatus *workflows.JobStatus) error {
	e.log.Info("Executing standard job", "job", jobName)

	if e.job == nil || len(e.job.Steps) == 0 {
		return fmt.Errorf("job has no steps defined")
	}

	// Create job execution context
	jobCtx, err := NewJobExecutionContext(client, wf, jobName, jobStatus, e.scheme, e.log, e.manager)
	if err != nil {
		return fmt.Errorf("error creating new job execution context: %w", err)
	}

	// Process each step sequentially
	for _, step := range e.job.Steps {
		if err := ExecuteStepWithContext(ctx, jobCtx, step, step.Name); err != nil {
			return err
		}
	}

	// All steps completed successfully
	return CompleteJob(jobStatus)
}

// CreateJobExecutor returns a JobExecutor based on which job configuration is set.
func CreateJobExecutor(job workflows.Job, scheme *runtime.Scheme, log logr.Logger, manager secretstore.ManagerInterface) (JobExecutor, error) {
	if job.Standard != nil {
		return NewStandardJobExecutor(job.Standard, scheme, log, manager), nil
	} else if job.Loop != nil {
		return NewLoopJobExecutor(job.Loop, scheme, log, manager), nil
	} else if job.Switch != nil {
		return NewSwitchJobExecutor(job.Switch, scheme, log, manager), nil
	}

	return nil, fmt.Errorf("no job configuration found")
}
