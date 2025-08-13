/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package secretstore

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	ctrlmetrics "github.com/external-secrets/external-secrets/pkg/controllers/metrics"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/tmetrics"

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
	if apierrors.IsNotFound(err) {
		tmetrics.RemoveMetrics(req.Namespace, req.Name)
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "unable to get Target")
		return ctrl.Result{}, err
	}

	return reconcile(ctx, req, genericStore, r.Client, log, Opts{
		ControllerClass: r.ControllerClass,
		GaugeVecGetter:  tmetrics.GetGaugeVec,
		Recorder:        r.recorder,
		RequeueInterval: r.RequeueInterval,
	})
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *TargetReconciler) SetupWithManager(mgr ctrl.Manager, obj client.Object, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("target")
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(obj).
		Complete(r)
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
