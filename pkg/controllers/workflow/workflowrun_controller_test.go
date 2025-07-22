// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package workflow

import (
	"context"
	"encoding/json"
	"testing"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	targetsv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type WorkflowRunReconcilerTestSuite struct {
	suite.Suite
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	builder  *fake.ClientBuilder
}

func (s *WorkflowRunReconcilerTestSuite) SetupTest() {
	s.scheme = runtime.NewScheme()
	workflows.AddToScheme(s.scheme)
	scanv1alpha1.AddToScheme(s.scheme)
	targetsv1alpha1.AddToScheme(s.scheme)
	corev1.AddToScheme(s.scheme)

	s.recorder = record.NewFakeRecorder(20)
	s.builder = fake.NewClientBuilder().WithScheme(s.scheme)
}

func (s *WorkflowRunReconcilerTestSuite) TestReconcileWorkflowCreated() {
	template := &workflows.WorkflowTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "WorkflowTemplate",
			APIVersion: "workflows.external-secrets.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: "default",
		},
		Spec: workflows.WorkflowTemplateSpec{
			Version: "v1",
			Name:    "Sample Workflow Template",
			Jobs: map[string]workflows.Job{
				"start": {
					Standard: &workflows.StandardJob{
						Steps: []workflows.Step{
							{
								Name: "step1",
								Debug: &workflows.DebugStep{
									Message: "Starting workflow",
								},
							},
						},
					},
				},
			},
		},
	}

	run := &workflows.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testrun",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "WorkflowRun",
			APIVersion: "workflows.external-secrets.io/v1alpha1",
		},
		Spec: workflows.WorkflowRunSpec{
			TemplateRef: workflows.TemplateRef{
				Name: "test-template",
			},
		},
	}

	cl := s.builder.WithObjects(template, run).WithStatusSubresource(template, run).Build()
	reconciler := &WorkflowRunReconciler{
		Client:   cl,
		Log:      logr.Discard(),
		Scheme:   s.scheme,
		Recorder: s.recorder,
	}

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(run),
	})

	require.NoError(s.T(), err)

	updatedRun := &workflows.WorkflowRun{}
	err = cl.Get(context.Background(), client.ObjectKeyFromObject(run), updatedRun)
	require.NoError(s.T(), err)
	assert.Len(s.T(), updatedRun.Status.Conditions, 1)
	assert.Equal(s.T(), "WorkflowCreated", updatedRun.Status.Conditions[0].Type)
	assert.Equal(s.T(), metav1.ConditionTrue, updatedRun.Status.Conditions[0].Status)
}

func (s *WorkflowRunReconcilerTestSuite) TestReconcileTemplateNotFound() {
	run := &workflows.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "testrun",
			Namespace:       "default",
			ResourceVersion: "1",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "WorkflowRun",
			APIVersion: "workflows.external-secrets.io/v1alpha1",
		},
		Spec: workflows.WorkflowRunSpec{
			TemplateRef: workflows.TemplateRef{
				Name: "nonexistent-template",
			},
		},
	}

	cl := s.builder.WithObjects(run).WithStatusSubresource(run).Build()
	reconciler := &WorkflowRunReconciler{
		Client:   cl,
		Log:      logr.Discard(),
		Scheme:   s.scheme,
		Recorder: s.recorder,
	}

	_, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(run),
	})

	require.NoError(s.T(), err)

	updatedRun := &workflows.WorkflowRun{}
	err = cl.Get(context.Background(), client.ObjectKeyFromObject(run), updatedRun)
	require.NoError(s.T(), err)
	assert.Len(s.T(), updatedRun.Status.Conditions, 1)
	assert.Equal(s.T(), "TemplateFound", updatedRun.Status.Conditions[0].Type)
	assert.Equal(s.T(), metav1.ConditionFalse, updatedRun.Status.Conditions[0].Status)
}

func (s *WorkflowRunReconcilerTestSuite) TestResolveWorkflowFromTemplateFindingArray() {
	// Simulate a Finding resource that returns a location
	finding := &scanv1alpha1.Finding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "finding1",
			Namespace: "default",
		},
		Status: scanv1alpha1.FindingStatus{
			Locations: []targetsv1alpha1.SecretInStoreRef{{
				Name:       "secret-store",
				Kind:       "SecretStore",
				APIVersion: "external-secrets.io/v1",
				RemoteRef: targetsv1alpha1.RemoteRef{
					Key:      "secret-key",
					Property: "secret-property",
				},
			}},
		},
	}

	param := workflows.Parameter{
		Name: "param1",
		Type: workflows.ParameterTypeFindingArray,
	}

	template := &workflows.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "template1",
			Namespace: "default",
		},
		Spec: workflows.WorkflowTemplateSpec{
			Version: "v1",
			Name:    "test-template",
			ParameterGroups: []workflows.ParameterGroup{
				{Parameters: []workflows.Parameter{param}},
			},
		},
	}

	// Arguments: simulate passing a finding array parameter
	argsJSON, _ := json.Marshal(map[string]any{
		"param1": []map[string]string{
			{"name": "finding1"},
		},
	})

	run := &workflows.WorkflowRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run1",
			Namespace: "default",
		},
		Spec: workflows.WorkflowRunSpec{
			TemplateRef: workflows.TemplateRef{Name: "template1"},
			Arguments: apiextensionsv1.JSON{
				Raw: argsJSON,
			},
		},
	}

	cl := s.builder.WithObjects(finding).Build()
	reconciler := &WorkflowRunReconciler{
		Client:   cl,
		Log:      logr.Discard(),
		Scheme:   s.scheme,
		Recorder: s.recorder,
	}

	workflow, err := reconciler.resolveWorkflowFromTemplate(context.Background(), template, run)
	require.NoError(s.T(), err)
	assert.NotNil(s.T(), workflow)
	assert.Contains(s.T(), string(workflow.Spec.Variables.Raw), "secret-store")
}

func TestWorkflowRunReconcilerTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowRunReconcilerTestSuite))
}
