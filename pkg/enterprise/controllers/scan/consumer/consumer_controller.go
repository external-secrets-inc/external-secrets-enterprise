// Copyright External Secrets Inc. 2025
// All Rights Reserved

package consumer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	targetv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
)

type ConsumerController struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (c *ConsumerController) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	consumer := &scanv1alpha1.Consumer{}
	if err := c.Get(ctx, req.NamespacedName, consumer); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	gvk := schema.GroupVersionKind{Group: targetv1alpha1.Group, Version: targetv1alpha1.Version, Kind: consumer.Spec.Type}
	obj, err := c.Scheme.New(gvk)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to create object %v: %w", gvk, err)
	}
	genericTarget, ok := obj.(targetv1alpha1.GenericTarget)
	if !ok {
		return ctrl.Result{}, fmt.Errorf("invalid object: %T", obj)
	}

	key := types.NamespacedName{
		Namespace: consumer.Spec.Target.Namespace,
		Name:      consumer.Spec.Target.Name,
	}
	if err := c.Get(ctx, key, genericTarget); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	status := genericTarget.GetTargetStatus()

	return c.CheckConsumerStatus(ctx, consumer, status.PushIndex)
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *ConsumerController) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&v1alpha1.Consumer{}).
		Complete(c)
}

func (c *ConsumerController) CheckConsumerStatus(ctx context.Context, consumer *scanv1alpha1.Consumer, pushSecretIndex map[string][]targetv1alpha1.SecretUpdateRecord) (ctrl.Result, error) {
	consumerStatusCondition := metav1.ConditionTrue
	consumerStatusReason := scanv1alpha1.ConsumerLocationsUpToDate
	consumerStatusMessage := "All observed locations are up to date"

	locationsOutOfDate := make([]string, 0)
	locationsOutOfDateMessages := make([]string, 0)

	for observedIndexKey, observedIndex := range consumer.Status.ObservedIndex {
		secretUpdateRecords, ok := pushSecretIndex[observedIndexKey]
		if !ok || len(secretUpdateRecords) == 0 {
			continue
		}

		latestRecord := secretUpdateRecords[len(secretUpdateRecords)-1]
		if latestRecord.SecretHash != observedIndex.SecretHash {
			locationsOutOfDate = append(locationsOutOfDate, observedIndexKey)
			locationsOutOfDateMessages = append(locationsOutOfDateMessages, fmt.Sprintf("Location %s last updated at %v. Current version updated at %v", observedIndexKey, observedIndex.Timestamp.Time, latestRecord.Timestamp.Time))
		}
	}

	if len(locationsOutOfDate) > 0 {
		consumerStatusCondition = metav1.ConditionFalse
		consumerStatusReason = scanv1alpha1.ConsumerLocationsOutOfDate
		consumerStatusMessage = fmt.Sprint("Observed locations out of date: ", strings.Join(locationsOutOfDateMessages, "; "))
	}

	for _, pods := range consumer.Status.Pods {
		if pods.Phase != "Running" {
			consumerStatusCondition = metav1.ConditionFalse
			consumerStatusReason = scanv1alpha1.ConsumerPodsNotReady
			consumerStatusMessage = "Not all pods related to this consumer are 'Running'"
			break
		}
	}

	changed := meta.SetStatusCondition(&consumer.Status.Conditions, metav1.Condition{
		Type:    string(scanv1alpha1.ConsumerLatestVersion),
		Status:  consumerStatusCondition,
		Reason:  consumerStatusReason,
		Message: consumerStatusMessage,
	})
	if changed {
		err := c.Status().Update(ctx, consumer)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update consumer status: %w", err)
		}
	}

	if consumerStatusCondition == metav1.ConditionFalse {
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}
