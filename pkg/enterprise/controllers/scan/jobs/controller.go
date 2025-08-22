// Copyright External Secrets Inc. 2025
// All Rights Reserved

package jobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"

	"github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	utils "github.com/external-secrets/external-secrets/pkg/enterprise/scan/jobs"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	_ "github.com/external-secrets/external-secrets/pkg/enterprise/targets/register"
)

type JobController struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	running sync.Map
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
	if jobSpec.Status.RunStatus != v1alpha1.JobRunStatusRunning {
		// Ignore new Runs
		if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyOnce {
			return ctrl.Result{}, nil
		}
		// TODO: add correct On Change condition
		if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyOnChange {
			return ctrl.Result{}, nil
		}

		if jobSpec.Spec.RunPolicy == v1alpha1.JobRunPolicyPull {
			// Check if a dependency has changed by comparing digests
			stores := &esv1.SecretStoreList{}
			if err := c.Client.List(ctx, stores, client.InNamespace(jobSpec.Namespace)); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to list secret stores for digest calculation: %w", err)
			}
			currentDigest := calculateDigest(stores.Items)

			// If digests are different, a SecretStore has changed, so run immediately.
			if currentDigest != jobSpec.Status.ObservedSecretStoresDigest {
				c.Log.V(1).Info("secretstore digest changed, running job immediately", "job", jobSpec.GetName())
			} else {
				// Otherwise, respect the polling interval
				timeToReconcile := time.Since(jobSpec.Status.LastRunTime.Time)
				if timeToReconcile < jobSpec.Spec.Interval.Duration {
					return ctrl.Result{RequeueAfter: jobSpec.Spec.Interval.Duration - timeToReconcile}, nil
				}
			}
		}
	}

	runningTime := time.Since(jobSpec.Status.LastRunTime.Time)
	timeout := jobSpec.Spec.JobTimeout.Duration

	if timeout > 0 && jobSpec.Status.RunStatus == v1alpha1.JobRunStatusRunning {
		if runningTime > timeout {
			c.stopJob(req)

			jobSpec.Status.RunStatus = v1alpha1.JobRunStatusFailed
			condition := metav1.Condition{
				Type:               string(v1alpha1.JobRunStatusFailed),
				Status:             metav1.ConditionFalse,
				Reason:             "TimedOut",
				Message:            fmt.Sprintf("timed out after %s", timeout),
				LastTransitionTime: metav1.Now(),
			}
			jobSpec.Status.Conditions = append(jobSpec.Status.Conditions, condition)
			jobSpec.Status.LastRunTime = metav1.Now()

			if err := c.Status().Update(ctx, jobSpec); err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{RequeueAfter: time.Second}, nil
		}

		// still running, requeue exactly when timeout would occur
		remaining := timeout - runningTime
		if remaining < time.Second {
			remaining = time.Second
		}
		return ctrl.Result{RequeueAfter: remaining}, nil
	}

	// Synchronize
	j := utils.NewJobRunner(c.Client, c.Log, jobSpec.Namespace, jobSpec.Spec.Constraints)

	jobSpec.Status = v1alpha1.JobStatus{
		LastRunTime: metav1.Now(),
		RunStatus:   v1alpha1.JobRunStatusRunning,
	}
	if err := c.Status().Update(ctx, jobSpec); err != nil {
		return ctrl.Result{}, err
	}

	// Start async job with cancel support
	runCtx, cancel := context.WithCancel(context.Background())
	c.running.Store(keyFor(req), cancel)

	// Run the Job applying constraints after leaving the reconcile loop
	defer func() {
		go func() {
			c.Log.V(1).Info("Starting async job", "job", jobSpec.GetName())
			defer c.running.Delete(keyFor(req))
			defer func() {
				_ = j.Close(context.Background())
			}()

			err := c.runJob(runCtx, jobSpec, j)
			if err != nil {
				c.Log.Error(err, "failed to run job")
			}
		}()
	}()

	if jobSpec.Spec.RunPolicy != v1alpha1.JobRunPolicyPull {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{RequeueAfter: jobSpec.Spec.Interval.Duration}, nil
}

func findingNeedsToUpdate(existing, finding *v1alpha1.Finding) bool {
	if existing == nil {
		return true
	}
	if finding == nil {
		return true
	}
	loc1 := existing.Status.Locations
	loc2 := finding.Status.Locations

	return !(slices.EqualFunc(loc1, loc2, utils.EqualLocations) && finding.Spec.Hash == existing.Spec.Hash)
}

func consumerNeedsToUpdate(existing, consumer *v1alpha1.Consumer) bool {
	if existing == nil {
		return true
	}
	if consumer == nil {
		return true
	}
	loc1 := existing.Status.Locations
	loc2 := consumer.Status.Locations

	return !(slices.EqualFunc(loc1, loc2, utils.EqualLocations) && slices.Equal(existing.Status.Conditions, consumer.Status.Conditions) && slices.Equal(existing.Status.Pods, consumer.Status.Pods))
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *JobController) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&v1alpha1.Job{}).
		Watches(
			&esv1.SecretStore{},
			handler.EnqueueRequestsFromMapFunc(c.mapSecretStoreToJobs),
		).
		Complete(c)
}

func (c *JobController) mapSecretStoreToJobs(ctx context.Context, obj client.Object) []reconcile.Request {
	c.Log.V(1).Info("reconciling all jobs due to SecretStore change", "secretstore", obj.GetName())

	jobList := &v1alpha1.JobList{}
	if err := c.List(ctx, jobList, client.InNamespace(obj.GetNamespace())); err != nil {
		c.Log.Error(err, "failed to list jobs for secretstore change")
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(jobList.Items))
	for i, job := range jobList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      job.Name,
				Namespace: job.Namespace,
			},
		}
	}
	return requests
}

// calculateDigest computes a sha256 digest from the resourceVersions of the provided SecretStores.
func calculateDigest(stores []esv1.SecretStore) string {
	if len(stores) == 0 {
		return ""
	}
	// Sort by name to ensure consistent digest
	sort.Slice(stores, func(i, j int) bool {
		return stores[i].Name < stores[j].Name
	})
	hash := sha256.New()
	for _, store := range stores {
		hash.Write([]byte(store.ResourceVersion))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (c *JobController) runJob(ctx context.Context, jobSpec *v1alpha1.Job, j *utils.JobRunner) error {
	defer func() {
		err := j.Close(context.Background())
		if err != nil {
			c.Log.Error(err, "failed to close job runner")
		}
	}()
	var jobTime metav1.Time
	var jobStatus v1alpha1.JobRunStatus
	var observedSecretStoresDigest string
	defer func() {
		jobSpec.Status = v1alpha1.JobStatus{
			LastRunTime:                jobTime,
			RunStatus:                  jobStatus,
			ObservedSecretStoresDigest: observedSecretStoresDigest,
		}
		c.Log.V(1).Info("Updating Job Status", "RunStatus", jobStatus)
		if err := c.Status().Update(ctx, jobSpec); err != nil {
			c.Log.Error(err, "failed to update job status")
		}
	}()
	c.Log.V(1).Info("Running Job", "job", jobSpec.GetName())
	findings, consumers, usedStores, err := j.Run(ctx)
	if err != nil {
		jobStatus = v1alpha1.JobRunStatusFailed
		jobTime = metav1.Now()
		return err
	}

	jobStatus, jobTime, err = c.UpdateFindings(ctx, findings, jobSpec.Namespace)
	if err != nil {
		return err
	}

	jobStatus, jobTime, err = c.UpdateConsumers(ctx, consumers, jobSpec.Namespace)
	if err != nil {
		return err
	}

	jobStatus = v1alpha1.JobRunStatusSucceeded
	jobTime = metav1.Now()
	observedSecretStoresDigest = calculateDigest(usedStores)
	return nil
}

func (c *JobController) UpdateFindings(ctx context.Context, findings []v1alpha1.Finding, namespace string) (v1alpha1.JobRunStatus, metav1.Time, error) {
	c.Log.V(1).Info("Found findings for job", "total findings", len(findings))
	// for each finding, see if it already exists and update it if it does;
	currentFindings := &v1alpha1.FindingList{}
	c.Log.V(1).Info("Listing Current findings")
	if err := c.List(ctx, currentFindings, client.InNamespace(namespace)); err != nil {
		return v1alpha1.JobRunStatusFailed, metav1.Now(), err
	}
	c.Log.V(1).Info("Found Current findings", "total findings", len(currentFindings.Items))

	currentFindingsByID := map[string]*v1alpha1.Finding{}
	for i := range currentFindings.Items {
		f := &currentFindings.Items[i]
		id := f.Spec.ID
		if id == "" {
			continue
		} // legacy; can be handled separately
		currentFindingsByID[id] = f
	}

	newFindingsByHash := map[string]*v1alpha1.Finding{}
	for i := range findings {
		f := &findings[i]
		newFindingsByHash[f.Spec.Hash] = f
	}

	assigned := utils.AssignIDs(currentFindings.Items, findings, utils.JaccardParams{MinJaccard: 0.7, MinIntersection: 2})
	seenIDs := make(map[string]struct{}, len(assigned))

	for i, assignedFinding := range assigned {
		newFinding := newFindingsByHash[findings[i].Spec.Hash]
		newFinding.Spec.ID = assignedFinding.Spec.ID
		seenIDs[assignedFinding.Spec.ID] = struct{}{}

		if currentFinding, ok := currentFindingsByID[assignedFinding.Spec.ID]; ok {
			if !findingNeedsToUpdate(currentFinding, newFinding) {
				continue
			}
			// Update Finding
			currentFinding.Status.Locations = newFinding.Status.Locations
			c.Log.V(1).Info("Updating finding", "finding", currentFinding.Spec.ID)
			if err := c.Status().Update(ctx, currentFinding); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}

			currentFinding.Spec.Hash = newFinding.Spec.Hash
			if err := c.Update(ctx, currentFinding); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
		} else {
			// create new CR with stable name
			create := newFinding.DeepCopy()
			create.SetNamespace(namespace)
			c.Log.V(1).Info("Creating finding", "finding", create.GetName())
			if err := c.Create(ctx, create); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
			create.Status.Locations = newFinding.Status.Locations
			c.Log.V(1).Info("Updating finding status", "finding", create.GetName())
			if err := c.Status().Update(ctx, create); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
		}
	}

	// Delete Findings that are no longer found
	for id, currentFinding := range currentFindingsByID {
		if _, ok := seenIDs[id]; !ok {
			c.Log.V(1).Info("Deleting stale finding (not observed this run)", "id", id, "name", currentFinding.GetName())
			if err := c.Delete(ctx, currentFinding); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
		}
	}

	return v1alpha1.JobRunStatusRunning, metav1.Now(), nil
}

func (c *JobController) UpdateConsumers(ctx context.Context, consumers []v1alpha1.Consumer, namespace string) (v1alpha1.JobRunStatus, metav1.Time, error) {
	c.Log.V(1).Info("Found consumers for job", "total consumers", len(consumers))
	// for each consumer, see if it already exists and update it if it does;
	currentConsumers := &v1alpha1.ConsumerList{}
	c.Log.V(1).Info("Listing Current consumers")
	if err := c.List(ctx, currentConsumers, client.InNamespace(namespace)); err != nil {
		return v1alpha1.JobRunStatusFailed, metav1.Now(), err
	}
	c.Log.V(1).Info("Found Current consumers", "total consumers", len(currentConsumers.Items))

	currentConsumersByID := map[string]*v1alpha1.Consumer{}
	for i := range currentConsumers.Items {
		consumer := &currentConsumers.Items[i]
		currentConsumersByID[consumer.Spec.ID] = consumer
	}

	seenIDs := make(map[string]struct{}, len(currentConsumersByID))

	for i := range consumers {
		newConsumer := &consumers[i]
		seenIDs[newConsumer.Spec.ID] = struct{}{}

		// Current update assumes that the ID will be the same for every consumer
		if currentConsumer, ok := currentConsumersByID[newConsumer.Spec.ID]; ok {
			if !consumerNeedsToUpdate(currentConsumer, newConsumer) {
				continue
			}

			currentConsumer.Status = newConsumer.Status
			c.Log.V(1).Info("Updating consumer", "consumer", currentConsumer.Spec.ID)
			if err := c.Status().Update(ctx, currentConsumer); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
		} else {
			// create new CR with stable name
			create := newConsumer.DeepCopy()
			create.SetNamespace(namespace)
			c.Log.V(1).Info("Creating consumer", "consumer", create.GetName())
			if err := c.Create(ctx, create); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
			create.Status.Locations = newConsumer.Status.Locations
			c.Log.V(1).Info("Updating consumer status", "consumer", create.GetName())
			if err := c.Status().Update(ctx, create); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
		}
	}

	// Delete Consumers that are no longer found
	for id, currentConsumer := range currentConsumersByID {
		if _, ok := seenIDs[id]; !ok {
			c.Log.V(1).Info("Deleting stale consumer (not observed this run)", "id", id, "name", currentConsumer.GetName())
			if err := c.Delete(ctx, currentConsumer); err != nil {
				return v1alpha1.JobRunStatusFailed, metav1.Now(), err
			}
		}
	}
	return v1alpha1.JobRunStatusRunning, metav1.Now(), nil
}

func (c *JobController) stopJob(req ctrl.Request) {
	key := keyFor(req)
	if v, ok := c.running.LoadAndDelete(key); ok {
		v.(context.CancelFunc)()
	}
}

func keyFor(req ctrl.Request) string { return req.NamespacedName.String() }
