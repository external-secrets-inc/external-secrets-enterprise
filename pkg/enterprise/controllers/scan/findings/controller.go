// Copyright External Secrets Inc. 2025
// All Rights Reserved

package federation

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/license"
	"github.com/external-secrets/external-secrets/pkg/enterprise/license/feature"
)

type FindingController struct {
	client.Client
	Log     logr.Logger
	Scheme  *runtime.Scheme
	feature feature.Feature
}

func (c *FindingController) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	findingSpec := &v1alpha1.Finding{}
	if err := c.Get(ctx, req.NamespacedName, findingSpec); err != nil {
		return feature.UnregisterIfNotFound(c.feature, findingSpec, err)
	}
	if findingSpec.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}
	if err := feature.RegisterOrFail(c.feature, findingSpec); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *FindingController) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	feat := feature.NewFeature("scan.findings", "Scan findings controller")
	if err := license.Register(feat); err != nil {
		return err
	}
	c.feature = feat
	if feat.IsAvailable() {
		return ctrl.NewControllerManagedBy(mgr).
			WithOptions(opts).
			For(&v1alpha1.Finding{}).
			Complete(c)
	}
	return nil
}
