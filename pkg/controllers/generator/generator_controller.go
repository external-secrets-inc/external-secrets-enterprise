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

package generator

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/license"
	"github.com/external-secrets/external-secrets/pkg/enterprise/license/feature"
)

var (
	GeneratorGroup   = "generators.external-secrets.io"
	GeneratorVersion = "v1alpha1"
)

type Reconciler struct {
	client.Client

	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	recorder   record.EventRecorder
	feature    feature.Feature

	Kind string
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	generic_generator, err := BuildGeneratorObject(r.Scheme, r.Kind)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error building generator object: %w", err)
	}

	if err := r.Get(ctx, req.NamespacedName, generic_generator); err != nil {
		return feature.UnregisterIfNotFound(r.feature, generic_generator, err)
	}
	if err := feature.RegisterOrFail(r.feature, generic_generator); err != nil {
		return ctrl.Result{}, err
	}
	gvk := generic_generator.GetObjectKind().GroupVersionKind()

	generator, found := genv1alpha1.GetGeneratorByKind(gvk.Kind)
	if !found {
		return ctrl.Result{}, fmt.Errorf("generator of kind %s not found", gvk.Kind)
	}

	err = generic_generator.SetOutputs(generator.GetKeys())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error setting outputs: %w", err)
	}

	err = r.Status().Update(ctx, generic_generator)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating generic generator: %w", err)
	}

	return ctrl.Result{}, nil
}

// mapKindToFeatureName converts generator Kind to the exact feature name used in trial license
func mapKindToFeatureName(kind string) string {
	// Map generator kinds to their exact trial license subscription names
	kindToSubscription := map[string]string{
		"AWSIam":     "aws_iam",
		"BasicAuth":  "basic_auth",
		"PostgreSQL": "postgresql",
		"Federation": "federation",
		"MongoDB":    "mongodb",
		"RabbitMQ":   "rabbitmq",
		"Neo4j":      "neo4j",
		"OpenAI":     "openai",
		"SSH":        "ssh",
		"SendGrid":   "sendgrid",
	}

	if subscriptionName, exists := kindToSubscription[kind]; exists {
		return fmt.Sprintf("generator.%s", subscriptionName)
	}
	// Fallback to lowercase conversion for unknown kinds
	return fmt.Sprintf("generator.%s", strings.ToLower(kind))
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, obj client.Object, opts controller.Options) error {
	// Create appropriate feature name based on Kind
	featureName := mapKindToFeatureName(r.Kind)
	feat := feature.NewFeature(featureName, fmt.Sprintf("%s Generator controller", r.Kind))
	if err := license.Register(feat); err != nil {
		return err
	}
	r.feature = feat
	if feat.IsAvailable() {
		r.recorder = mgr.GetEventRecorderFor("external-secrets")
		return ctrl.NewControllerManagedBy(mgr).
			WithOptions(opts).
			For(obj).
			Complete(r)
	}
	return nil
}

func BuildGeneratorObject(scheme *runtime.Scheme, kind string) (genv1alpha1.GenericGenerator, error) {
	gvk := schema.GroupVersionKind{Group: GeneratorGroup, Version: GeneratorVersion, Kind: kind}
	obj, err := scheme.New(gvk)
	if err != nil {
		return nil, fmt.Errorf("failed to create object %v: %w", gvk, err)
	}
	co, ok := obj.(genv1alpha1.GenericGenerator)
	if !ok {
		return nil, fmt.Errorf("invalid object: %T", obj)
	}
	return co, nil
}
