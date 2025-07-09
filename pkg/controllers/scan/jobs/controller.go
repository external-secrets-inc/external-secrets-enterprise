// Copyright External Secrets Inc. 2025
// All Rights Reserved

package jobs

import (
	"context"
	"slices"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	utils "github.com/external-secrets/external-secrets/pkg/scan/jobs"
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
	//TODO: Add ShouldReconcile Method checking if Job already has ran at least once

	// Synchronize
	j := utils.NewJobRunner(c.Client, c.Log, jobSpec.Namespace, jobSpec.Spec.Constraints)
	// Run the Job applying constraints
	findings, err := j.Run(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
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
				if err := c.Create(ctx, create); err != nil {
					return ctrl.Result{}, err
				}
				create.Status.Locations = finding.Status.Locations
				if err := c.Status().Update(ctx, create); err != nil {
					return ctrl.Result{}, err
				}
			} else {
				return ctrl.Result{}, err
			}
		} else {
			if needsToUpdate(existing, &finding) {
				existing.Status.Locations = finding.Status.Locations
				if err := c.Status().Update(ctx, existing); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}
	jobSpec.Status = v1alpha1.JobStatus{
		LastRunTime: metav1.Now(),
		RunStatus:   v1alpha1.JobRunStatusSucceeded,
	}
	if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyOnce {
		return ctrl.Result{}, nil
	}
	if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyOnChange {
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
	return !slices.EqualFunc(loc1, loc2, func(a, b v1alpha1.SecretInStoreRef) bool {
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
