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
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// WorkflowRunReconciler reconciles a WorkflowRun object.
type WorkflowRunReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowruns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflowtemplates,verbs=get;list;watch
//+kubebuilder:rbac:groups=workflows.external-secrets.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles WorkflowRun resources.
func (r *WorkflowRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("workflowrun", req.NamespacedName)
	log.Info("reconciling WorkflowRun")

	// Fetch the WorkflowRun instance
	run := &workflows.WorkflowRun{}
	if err := r.Get(ctx, req.NamespacedName, run); err != nil {
		// We'll ignore not-found errors, since they can't be fixed by an immediate requeue
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If workflow already created, check its status
	if run.Status.WorkflowRef != nil {
		return r.checkWorkflowStatus(ctx, run)
	}

	// Fetch the template
	template := &workflows.WorkflowTemplate{}
	templateNamespace := run.Spec.TemplateRef.Namespace
	if templateNamespace == "" {
		templateNamespace = run.Namespace
	}

	if err := r.Get(ctx, types.NamespacedName{
		Name:      run.Spec.TemplateRef.Name,
		Namespace: templateNamespace,
	}, template); err != nil {
		if errors.IsNotFound(err) {
			r.Recorder.Event(run, corev1.EventTypeWarning, "TemplateNotFound",
				fmt.Sprintf("Template %s not found in namespace %s", run.Spec.TemplateRef.Name, templateNamespace))
			// Update status with error condition
			run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
				Type:               "TemplateFound",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "TemplateNotFound",
				Message:            fmt.Sprintf("Template %s not found in namespace %s", run.Spec.TemplateRef.Name, templateNamespace),
			})
			if err := r.Status().Update(ctx, run); err != nil {
				log.Error(err, "unable to update WorkflowRun status")
				return ctrl.Result{}, err
			}
			// Requeue after some time in case the workflow template is created later
			return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
		}
		log.Error(err, "unable to fetch WorkflowTemplate")
		return ctrl.Result{}, err
	}

	// Create workflow from template
	workflow, err := r.resolveWorkflowFromTemplate(template, run)
	if err != nil {
		r.Recorder.Event(run, corev1.EventTypeWarning, "ResolutionFailed",
			fmt.Sprintf("Failed to resolve workflow from template: %v", err))
		// Update status with error condition
		run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
			Type:               "WorkflowResolved",
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "ResolutionFailed",
			Message:            fmt.Sprintf("Failed to resolve workflow from template: %v", err),
		})
		if err := r.Status().Update(ctx, run); err != nil {
			log.Error(err, "unable to update WorkflowRun status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	// Set the WorkflowRun as the owner of the Workflow
	if err := controllerutil.SetControllerReference(run, workflow, r.Scheme); err != nil {
		log.Error(err, "unable to set controller reference on Workflow")
		return ctrl.Result{}, err
	}

	// Create the workflow
	if err := r.Create(ctx, workflow); err != nil {
		log.Error(err, "unable to create Workflow for WorkflowRun")
		return ctrl.Result{}, err
	}

	// Update run status with workflow reference
	run.Status.WorkflowRef = &workflows.WorkflowRef{
		Name:      workflow.Name,
		Namespace: workflow.Namespace,
	}

	// Add success condition
	run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
		Type:               "WorkflowCreated",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "WorkflowCreated",
		Message:            fmt.Sprintf("Created workflow %s from template %s", workflow.Name, template.Name),
	})

	// Update status
	if err := r.Status().Update(ctx, run); err != nil {
		log.Error(err, "unable to update WorkflowRun status")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(run, corev1.EventTypeNormal, "WorkflowCreated",
		fmt.Sprintf("Created workflow %s from template %s", workflow.Name, template.Name))

	return ctrl.Result{}, nil
}

// checkWorkflowStatus checks the status of the created workflow and updates the WorkflowRun status accordingly.
func (r *WorkflowRunReconciler) checkWorkflowStatus(ctx context.Context, run *workflows.WorkflowRun) (ctrl.Result, error) {
	log := r.Log.WithValues("workflowrun", types.NamespacedName{Name: run.Name, Namespace: run.Namespace})

	// Fetch the workflow
	workflow := &workflows.Workflow{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      run.Status.WorkflowRef.Name,
		Namespace: run.Status.WorkflowRef.Namespace,
	}, workflow); err != nil {
		if errors.IsNotFound(err) {
			// Workflow was deleted, update the status
			run.Status.Conditions = append(run.Status.Conditions, metav1.Condition{
				Type:               "WorkflowExists",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
				Reason:             "WorkflowDeleted",
				Message:            fmt.Sprintf("Workflow %s was deleted", run.Status.WorkflowRef.Name),
			})
			if err := r.Status().Update(ctx, run); err != nil {
				log.Error(err, "unable to update WorkflowRun status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch Workflow")
		return ctrl.Result{}, err
	}

	// Check if the workflow status has changed
	// TODO: Probably need to check for all of the possible status values for the workflow
	statusChanged := false
	for _, cond := range workflow.Status.Conditions {
		// Check if this condition already exists in the WorkflowRun
		exists := false
		for _, runCond := range run.Status.Conditions {
			if runCond.Type == cond.Type && runCond.Status == cond.Status && runCond.Reason == cond.Reason {
				exists = true
				break
			}
		}

		if !exists {
			run.Status.Conditions = append(run.Status.Conditions, cond)
			statusChanged = true
		}
	}

	if statusChanged {
		if err := r.Status().Update(ctx, run); err != nil {
			log.Error(err, "unable to update WorkflowRun status")
			return ctrl.Result{}, err
		}
	}

	// Requeue to check for status updates
	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

// resolveWorkflowFromTemplate creates a new Workflow from a WorkflowTemplate and WorkflowRun.
func (r *WorkflowRunReconciler) resolveWorkflowFromTemplate(template *workflows.WorkflowTemplate, run *workflows.WorkflowRun) (*workflows.Workflow, error) {
	// Create a new workflow
	workflow := &workflows.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      run.Name,
			Namespace: run.Namespace,
			Labels: map[string]string{
				"workflows.external-secrets.io/template": template.Name,
				"workflows.external-secrets.io/run":      run.Name,
			},
		},
		Spec: workflows.WorkflowSpec{
			Version:   template.Spec.Version,
			Name:      template.Spec.Name,
			Variables: make(map[string]string),
			Jobs:      template.Spec.Jobs,
		},
	}

	// Convert arguments to variables
	for _, param := range template.Spec.Parameters {
		value, exists := run.Spec.Arguments[param.Name]
		if !exists {
			if param.Required && param.Default == "" {
				return nil, fmt.Errorf("required parameter %s not provided", param.Name)
			}
			if param.Default != "" {
				workflow.Spec.Variables[param.Name] = param.Default
			}
		} else {
			workflow.Spec.Variables[param.Name] = value
		}
	}

	return workflow, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&workflows.WorkflowRun{}).
		Owns(&workflows.Workflow{}).
		Complete(r)
}
