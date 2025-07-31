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
)

type FindingController struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (c *FindingController) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	// Placeholder if we need to implement a controller for findings.
	// Still unclear if we do
	return ctrl.Result{}, nil
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *FindingController) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&v1alpha1.Finding{}).
		Complete(c)
}
