// Copyright External Secrets Inc. 2025
// All Rights Reserved

package target

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/enterprise/controllers/target/tmetrics"
	"github.com/external-secrets/external-secrets/pkg/enterprise/license"
	"github.com/external-secrets/external-secrets/pkg/enterprise/license/feature"

	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/provider/register"
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

const (
	TargetGroup   = "target.external-secrets.io"
	TargetVersion = "v1alpha1"
)

// TargetReconciler reconciles a Target object.
type TargetReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	RequeueInterval time.Duration
	ControllerClass string
	feature         feature.Feature

	Kind string
}

func (r *TargetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("target", req.NamespacedName)

	resourceLabels := ctrlmetrics.RefineNonConditionMetricLabels(map[string]string{"name": req.Name, "namespace": req.Namespace})
	start := time.Now()

	secretStoreReconcileDuration := tmetrics.GetGaugeVec(tmetrics.TargetReconcileDurationKey)
	defer func() { secretStoreReconcileDuration.With(resourceLabels).Set(float64(time.Since(start))) }()

	genericStore, err := BuildTargetObject(r.Scheme, r.Kind)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error building generator object: %w", err)
	}

	err = r.Get(ctx, req.NamespacedName, genericStore)
	if err != nil {
		log.Error(err, "unable to get Target")
		if apierrors.IsNotFound(err) {
			tmetrics.RemoveMetrics(req.Namespace, req.Name)
		}
		return feature.UnregisterIfNotFound(r.feature, genericStore, err)
	}
	if err := feature.RegisterOrFail(r.feature, genericStore); err != nil {
		return ctrl.Result{}, err
	}

	return secretstore.Reconcile(ctx, req, genericStore, r.Client, log, secretstore.Opts{
		ControllerClass: r.ControllerClass,
		GaugeVecGetter:  tmetrics.GetGaugeVec,
		Recorder:        r.recorder,
		RequeueInterval: r.RequeueInterval,
	})
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *TargetReconciler) SetupWithManager(mgr ctrl.Manager, obj client.Object, opts controller.Options) error {
	// Create appropriate feature name based on Kind
	featureName := mapKindToFeatureName(r.Kind)
	feat := feature.NewFeature(featureName, fmt.Sprintf("%s Generator controller", r.Kind))
	if err := license.Register(feat); err != nil {
		return err
	}
	r.feature = feat
	if feat.IsAvailable() {
		r.recorder = mgr.GetEventRecorderFor("target")
		return ctrl.NewControllerManagedBy(mgr).
			WithOptions(opts).
			For(obj).
			Complete(r)
	}
	return nil
}

func BuildTargetObject(scheme *runtime.Scheme, kind string) (esv1.GenericStore, error) {
	gvk := schema.GroupVersionKind{Group: TargetGroup, Version: TargetVersion, Kind: kind}
	obj, err := scheme.New(gvk)
	if err != nil {
		return nil, fmt.Errorf("failed to create object %v: %w", gvk, err)
	}
	co, ok := obj.(esv1.GenericStore)
	if !ok {
		return nil, fmt.Errorf("invalid object: %T", obj)
	}
	return co, nil
}

// mapKindToFeatureName converts target Kind to the exact feature name used in trial license.
func mapKindToFeatureName(kind string) string {
	// Map target kinds to their exact trial license subscription names
	kindToSubscription := map[string]string{
		tgtv1alpha1.GithubTargetKind:         tgtv1alpha1.GithubTargetSubscriptionName,
		tgtv1alpha1.KubernetesTargetKind:     tgtv1alpha1.KubernetesTargetSubscriptionName,
		tgtv1alpha1.VirtualMachineTargetKind: tgtv1alpha1.VirtualMachineTargetSubscriptionName,
	}

	if subscriptionName, exists := kindToSubscription[kind]; exists {
		return subscriptionName
	}
	// Fallback to lowercase conversion for unknown kinds
	return fmt.Sprintf("target.%s", strings.ToLower(kind))
}
