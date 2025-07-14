// Copyright External Secrets Inc. 2025
// All Rights Reserved

package jobs

import (
	"context"
	"slices"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	utils "github.com/external-secrets/external-secrets/pkg/scan/jobs"

	_ "github.com/external-secrets/external-secrets/pkg/targets/register"
)

type JobController struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (c *JobController) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	jobSpec := &v1alpha1.Job{}
	if err := c.Get(ctx, req.NamespacedName, jobSpec); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if jobSpec.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}
	// Check if we should already run this job
	if jobSpec.Status.RunStatus == v1alpha1.JobRunStatusSucceeded {
		// Ignore new Runs
		if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyOnce {
			return ctrl.Result{}, nil
		}
		// TODO: add correct On Change condition
		if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyOnChange {
			return ctrl.Result{}, nil
		}
		if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyPull {
			timeToReconcile := time.Since(jobSpec.Status.LastRunTime.Time)
			if timeToReconcile < jobSpec.Spec.Interval.Duration {
				return ctrl.Result{RequeueAfter: jobSpec.Spec.Interval.Duration - timeToReconcile}, nil
			}
		}
	}
	//TODO: Add ShouldReconcile Method checking if Job already has ran at least once
	if jobSpec.Status.RunStatus == v1alpha1.JobRunStatusRunning {
		// Ignore because the job is still running - wait it to finish with the appropriate Update Call
		return ctrl.Result{}, nil
	}
	// Synchronize
	j := utils.NewJobRunner(c.Client, c.Log, jobSpec.Namespace, jobSpec.Spec.Constraints)
	// Run the Job applying constraints after leaving the reconcile loop
	defer func() {
		go func() {
			c.Log.V(1).Info("Starting async job", "job", jobSpec.GetName())
			err := c.runJob(context.Background(), jobSpec, j)
			if err != nil {
				c.Log.Error(err, "failed to run job")
			}
		}()
	}()
	jobSpec.Status = v1alpha1.JobStatus{
		LastRunTime: metav1.Now(),
		RunStatus:   v1alpha1.JobRunStatusRunning,
	}
	if err := c.Status().Update(ctx, jobSpec); err != nil {
		return ctrl.Result{}, err
	}
	if jobSpec.Spec.RunPolicy != v1alpha1.JobRunPolicyPull {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: jobSpec.Spec.Interval.Duration}, nil
}

func needsToUpdate(existing, finding *v1alpha1.Finding) bool {
	if existing == nil {
		return true
	}
	if finding == nil {
		return true
	}
	loc1 := existing.Status.Locations
	loc2 := finding.Status.Locations
	return !slices.EqualFunc(loc1, loc2, func(a, b tgtv1alpha1.SecretInStoreRef) bool {
		return a.Name == b.Name && a.Kind == b.Kind && a.APIVersion == b.APIVersion && a.RemoteRef.Key == b.RemoteRef.Key && a.RemoteRef.Property == b.RemoteRef.Property
	})
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *JobController) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&v1alpha1.Job{}).
		Complete(c)
}

func (c *JobController) runJob(ctx context.Context, jobSpec *v1alpha1.Job, j *utils.JobRunner) error {
	var jobTime metav1.Time
	var jobStatus v1alpha1.JobRunStatus
	defer func() {
		jobSpec.Status = v1alpha1.JobStatus{
			LastRunTime: jobTime,
			RunStatus:   jobStatus,
		}
		c.Log.V(1).Info("Updating Job Status", "RunStatus", jobStatus)
		if err := c.Status().Update(ctx, jobSpec); err != nil {
			c.Log.Error(err, "failed to update job status")
		}
	}()
	c.Log.V(1).Info("Running Job", "job", jobSpec.GetName())
	findings, err := j.Run(ctx)
	if err != nil {
		jobStatus = v1alpha1.JobRunStatusFailed
		jobTime = metav1.Now()
		return err
	}
	c.Log.V(1).Info("Found findings for job", "total findings", len(findings))
	// for each finding, see if it already exists and update it if it does;
	for _, finding := range findings {
		req := client.ObjectKey{
			Name:      finding.Name,
			Namespace: jobSpec.Namespace,
		}
		existing := &v1alpha1.Finding{}
		finding.SetNamespace(jobSpec.Namespace)
		if err := c.Get(ctx, req, existing); err != nil {
			if apierrors.IsNotFound(err) {
				// Create Finding
				create := finding.DeepCopy()
				c.Log.V(1).Info("Creating finding", "finding", create.GetName())
				if err := c.Create(ctx, create); err != nil {
					jobStatus = v1alpha1.JobRunStatusFailed
					jobTime = metav1.Now()
					return err
				}
				create.Status.Locations = finding.Status.Locations
				c.Log.V(1).Info("Updating finding status", "finding", create.GetName())
				if err := c.Status().Update(ctx, create); err != nil {
					jobStatus = v1alpha1.JobRunStatusFailed
					jobTime = metav1.Now()
					return err
				}
			} else {
				jobStatus = v1alpha1.JobRunStatusFailed
				jobTime = metav1.Now()
				return err
			}
		} else {
			if needsToUpdate(existing, &finding) {
				existing.Status.Locations = finding.Status.Locations
				c.Log.V(1).Info("Updating finding", "finding", existing.GetName())
				if err := c.Status().Update(ctx, existing); err != nil {
					jobStatus = v1alpha1.JobRunStatusFailed
					jobTime = metav1.Now()
					return err
				}
			}
		}
	}
	jobStatus = v1alpha1.JobRunStatusSucceeded
	jobTime = metav1.Now()
	return nil
}
