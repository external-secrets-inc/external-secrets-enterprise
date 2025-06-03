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

	"github.com/external-secrets/external-secrets/apis/federation/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/federation/provider"
	"github.com/external-secrets/external-secrets/pkg/federation/store"
)

// TODO - make this operate over all *.federation.external-secrets.io resources.
type SpiffeFederationController struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (c *SpiffeFederationController) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	// Get the Authorization.fedetarion.external-secrets.io object
	authorization := &v1alpha1.SpiffeFederation{}
	if err := c.Get(ctx, req.NamespacedName, authorization); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	ref := v1alpha1.FederationRef{
		Name: authorization.Name,
		Kind: "SpiffeFederation",
	}
	prov := provider.NewSpiffeProvider(authorization.Spec.TrustDomain)
	// Get the Spec and add it to the federation store
	store.AddStore(ref, prov)
	return ctrl.Result{}, nil
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (c *SpiffeFederationController) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.SpiffeFederation{}).
		Complete(c)
}
