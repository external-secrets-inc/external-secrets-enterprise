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
package workflow

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// WorkflowTemplateReconciler reconciles a WorkflowTemplate object.
type WorkflowTemplateReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowtemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowtemplates/status,verbs=get;update;patch

// Reconcile handles WorkflowTemplate resources.
func (r *WorkflowTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("workflowtemplate", req.NamespacedName)
	log.Info("reconciling WorkflowTemplate")

	// Fetch the WorkflowTemplate instance
	template := &workflows.WorkflowTemplate{}
	if err := r.Get(ctx, req.NamespacedName, template); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Validate the template
	if err := r.validateTemplate(template); err != nil {
		log.Error(err, "invalid template")
		r.Recorder.Event(template, "Warning", "ValidationFailed", fmt.Sprintf("Template validation failed: %v", err))
		return ctrl.Result{}, err
	}

	// Template is valid, nothing else to do
	r.Recorder.Event(template, "Normal", "Validated", "Template validation succeeded")
	return ctrl.Result{}, nil
}

// validateTemplate validates a WorkflowTemplate.
func (r *WorkflowTemplateReconciler) validateTemplate(template *workflows.WorkflowTemplate) error {
	// TODO: The same validation logic for the workflows needs to be applied here too - jobs, dependencies, etc.

	// Check that the template has a name
	if template.Spec.Name == "" {
		return fmt.Errorf("template must have a name")
	}

	// Check that the template has a version
	if template.Spec.Version == "" {
		return fmt.Errorf("template must have a version")
	}

	// Check that the template has at least one job
	if len(template.Spec.Jobs) == 0 {
		return fmt.Errorf("template must have at least one job")
	}

	// Validate parameters
	paramNames := make(map[string]bool)
	for _, param := range template.Spec.Parameters {
		// Check for duplicate parameter names
		if paramNames[param.Name] {
			return fmt.Errorf("duplicate parameter name: %s", param.Name)
		}
		paramNames[param.Name] = true

		// Check that required parameters don't have empty names
		if param.Name == "" {
			return fmt.Errorf("parameter must have a name")
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workflows.WorkflowTemplate{}).
		Complete(r)
}
