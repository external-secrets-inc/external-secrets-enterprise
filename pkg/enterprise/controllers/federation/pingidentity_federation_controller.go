// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package federation

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	idfedv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/federation/identity/v1alpha1"
	fedv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/federation/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/federation/provider"
	"github.com/external-secrets/external-secrets/pkg/enterprise/federation/store"
)

type PingIdentityFederationController struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (c *PingIdentityFederationController) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	pingidentity := &idfedv1alpha1.PingIdentityFederation{}
	if err := c.Get(ctx, req.NamespacedName, pingidentity); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create FederationRef for this PingIdentity federation
	ref := fedv1alpha1.FederationRef{
		Name: pingidentity.Name,
		Kind: "PingIdentityFederation",
	}

	// Create PingIdentity provider with configuration from Spec
	prov := provider.NewPingIdentityProvider(pingidentity.Spec.Region, pingidentity.Spec.EnvironmentID)

	// Register provider in the federation store
	store.AddStore(ref, prov)

	c.Log.Info("Registered PingIdentity federation provider",
		"name", pingidentity.Name,
		"region", pingidentity.Spec.Region,
		"environmentId", pingidentity.Spec.EnvironmentID)

	return ctrl.Result{}, nil
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *PingIdentityFederationController) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&idfedv1alpha1.PingIdentityFederation{}).
		Complete(c)
}
