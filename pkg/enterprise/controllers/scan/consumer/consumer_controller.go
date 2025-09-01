// Copyright External Secrets Inc. 2025
// All Rights Reserved

package consumer

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	var obj client.Object
	switch consumer.Spec.Type {
	case targetv1alpha1.GithubTargetKind:
		obj = &targetv1alpha1.GithubRepository{}
	case targetv1alpha1.KubernetesTargetKind:
		obj = &targetv1alpha1.KubernetesCluster{}
	case targetv1alpha1.VirtualMachineKind:
		obj = &targetv1alpha1.VirtualMachine{}
	default:
		return ctrl.Result{}, fmt.Errorf("unsupported target kind: %q", consumer.Spec.Type)
	}

	key := types.NamespacedName{
		Namespace: consumer.Spec.Target.Namespace,
		Name:      consumer.Spec.Target.Name,
	}
	if err := c.Get(ctx, key, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var pushSecretIndex map[string][]targetv1alpha1.SecretUpdateRecord
	switch t := obj.(type) {
	case *targetv1alpha1.GithubRepository:
		pushSecretIndex = t.Status.PushIndex
	case *targetv1alpha1.KubernetesCluster:
		pushSecretIndex = t.Status.PushIndex
	case *targetv1alpha1.VirtualMachine:
		pushSecretIndex = t.Status.PushIndex
	default:
		return ctrl.Result{}, fmt.Errorf("unexpected type %T", obj)
	}

	err = c.CheckConsumerStatus(ctx, consumer, pushSecretIndex)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check consumer status: %w", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *ConsumerController) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&v1alpha1.Consumer{}).
		Complete(c)
}

func (c *ConsumerController) CheckConsumerStatus(ctx context.Context, consumer *scanv1alpha1.Consumer, pushSecretIndex map[string][]targetv1alpha1.SecretUpdateRecord) error {
	consumerStatusType := scanv1alpha1.ConsumerLatestVersion
	consumerStatusReason := "LocationsUpToDate"
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
		consumerStatusType = scanv1alpha1.ConsumerPendingUpdate
		consumerStatusReason = "LocationsOutOfDate"
		consumerStatusMessage = fmt.Sprint("Observed locations out of date: ", strings.Join(locationsOutOfDateMessages, "; "))
	}

	changed := meta.SetStatusCondition(&consumer.Status.Conditions, metav1.Condition{
		Type:    string(consumerStatusType),
		Status:  metav1.ConditionTrue,
		Reason:  consumerStatusReason,
		Message: consumerStatusMessage,
	})
	if changed {
		err := c.Status().Update(ctx, consumer)
		if err != nil {
			return fmt.Errorf("failed to update consumer status: %w", err)
		}
	}
	return nil
}
