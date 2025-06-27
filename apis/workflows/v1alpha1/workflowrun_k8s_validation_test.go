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

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestValidateKubernetesResourceValidation(t *testing.T) {
	// Create a test scheme
	scheme := runtime.NewScheme()
	_ = AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = esv1.AddToScheme(scheme)

	// Create the test namespace
	testNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Create a test SecretStore
	testSecretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-store",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AWS: &esv1.AWSProvider{
					Region: "us-west-2",
				},
			},
		},
	}

	// Create a test SecretStore
	testSecondSecretStore := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-second-store",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AWS: &esv1.AWSProvider{
					Region: "us-west-2",
				},
			},
		},
	}

	// Create a test ClusterSecretStore
	testClusterSecretStore := &esv1.ClusterSecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster-store",
			Labels: map[string]string{
				"env": "test",
			},
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				AWS: &esv1.AWSProvider{
					Region: "us-east-1",
				},
			},
		},
	}

	// Create a test template with Kubernetes resource parameters
	template := &WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "k8s-resource-template",
			Namespace: "test-namespace",
		},
		Spec: WorkflowTemplateSpec{
			Version: "v1",
			Name:    "K8s Resource Test",
			ParameterGroups: []ParameterGroup{
				{
					Name: "Resources",
					Parameters: []Parameter{
						{
							Name:     "targetNamespace",
							Type:     ParameterTypeNamespace,
							Required: true,
						},
						{
							Name:     "secretStore",
							Type:     ParameterTypeSecretStore,
							Required: true,
							ResourceConstraints: &ResourceConstraints{
								Namespace: "test-namespace",
								LabelSelector: map[string]string{
									"app": "test",
								},
							},
						},
						{
							Name:     "secretStoreArray",
							Type:     ParameterTypeSecretStoreArray,
							Required: false,
						},
						{
							Name:     "clusterSecretStore",
							Type:     ParameterTypeClusterSecretStore,
							Required: false,
							ResourceConstraints: &ResourceConstraints{
								LabelSelector: map[string]string{
									"env": "test",
								},
							},
						},
						{
							Name:     "secretStoreNoCrossNS",
							Type:     ParameterTypeSecretStore,
							Required: false,
							ResourceConstraints: &ResourceConstraints{
								AllowCrossNamespace: false,
							},
						},
					},
				},
			},
			Jobs: map[string]Job{
				"test": {
					Standard: &StandardJob{
						Steps: []Step{
							{
								Name: "test",
								Debug: &DebugStep{
									Message: "Test",
								},
							},
						},
					},
				},
			},
		},
	}

	// Create a fake client with test objects
	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(testNamespace, testSecretStore, testSecondSecretStore, testClusterSecretStore, template).
		Build()

	// Set the validation client
	SetValidationClient(client)

	// Test cases
	tests := []struct {
		name        string
		workflowRun *WorkflowRun
		wantErr     bool
		errMsg      string
	}{
		{
			name: "valid secretStoreArray",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: map[string]string{
						"targetNamespace":  "test-namespace",
						"secretStore":      "test-store",
						"secretStoreArray": "test-store,test-second-store",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid secretStoreArray with one element",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: map[string]string{
						"targetNamespace":  "test-namespace",
						"secretStore":      "test-store",
						"secretStoreArray": "test-second-store",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid StoreArray",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: map[string]string{
						"targetNamespace":  "test-namespace",
						"secretStore":      "test-store",
						"secretStoreArray": "test-second-store,unexisting-store",
					},
				},
			},
			wantErr: true,
			errMsg:  "resource unexisting-store of type array[secretstore] not found in namespace test-namespace",
		},

		{
			name: "valid k8s resources",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: map[string]string{
						"targetNamespace":    "test-namespace",
						"secretStore":        "test-store",
						"clusterSecretStore": "test-cluster-store",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "non-existent namespace",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-ns-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: map[string]string{
						"targetNamespace": "non-existent-namespace",
						"secretStore":     "test-store",
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-namespace of type namespace not found",
		},
		{
			name: "non-existent secret store",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-store-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: map[string]string{
						"targetNamespace": "test-namespace",
						"secretStore":     "non-existent-store",
					},
				},
			},
			wantErr: true,
			errMsg:  "resource non-existent-store of type secretstore not found in namespace test-namespace",
		},
		{
			name: "secret store label selector mismatch",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "label-mismatch-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: map[string]string{
						"targetNamespace": "test-namespace",
						"secretStore":     "test-store",
					},
				},
			},
			wantErr: false, // This should pass because the test-store has the correct label
		},
		{
			name: "cluster secret store label selector mismatch",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-label-mismatch-run",
					Namespace: "test-namespace",
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name: "k8s-resource-template",
					},
					Arguments: map[string]string{
						"targetNamespace":    "test-namespace",
						"secretStore":        "test-store",
						"clusterSecretStore": "test-cluster-store",
					},
				},
			},
			wantErr: false, // This should pass because test-cluster-store has the correct label
		},
		{
			name: "cross-namespace not allowed",
			workflowRun: &WorkflowRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cross-ns-not-allowed-run",
					Namespace: "default", // Different namespace
				},
				Spec: WorkflowRunSpec{
					TemplateRef: TemplateRef{
						Name:      "k8s-resource-template",
						Namespace: "test-namespace",
					},
					Arguments: map[string]string{
						"targetNamespace":      "test-namespace",
						"secretStore":          "test-store",
						"secretStoreNoCrossNS": "test-store", // This should fail because it's in a different namespace
					},
				},
			},
			wantErr: true,
			errMsg:  "cross-namespace resource references are not allowed for this parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorkflowRunParameters(tt.workflowRun)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateKubernetesResourceTypes(t *testing.T) {
	// Test the parameter type helper methods
	tests := []struct {
		name          string
		paramType     ParameterType
		isK8sResource bool
		apiVersion    string
		kind          string
	}{
		{
			name:          "namespace type",
			paramType:     ParameterTypeNamespace,
			isK8sResource: true,
			apiVersion:    "v1",
			kind:          "Namespace",
		},
		{
			name:          "secretstore type",
			paramType:     ParameterTypeSecretStore,
			isK8sResource: true,
			apiVersion:    "external-secrets.io/v1",
			kind:          "SecretStore",
		},
		{
			name:          "clustersecretstore type",
			paramType:     ParameterTypeClusterSecretStore,
			isK8sResource: true,
			apiVersion:    "external-secrets.io/v1",
			kind:          "ClusterSecretStore",
		},
		{
			name:          "externalsecret type",
			paramType:     ParameterTypeExternalSecret,
			isK8sResource: true,
			apiVersion:    "external-secrets.io/v1",
			kind:          "ExternalSecret",
		},
		{
			name:          "generator type",
			paramType:     ParameterTypeGenerator,
			isK8sResource: true,
			apiVersion:    "v1alpha1",
			kind:          "Generator",
		},
		{
			name:          "string type",
			paramType:     ParameterTypeString,
			isK8sResource: false,
			apiVersion:    "",
			kind:          "",
		},
		{
			name:          "number type",
			paramType:     ParameterTypeNumber,
			isK8sResource: false,
			apiVersion:    "",
			kind:          "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isK8sResource, tt.paramType.IsKubernetesResource())
			if tt.isK8sResource {
				assert.Equal(t, tt.apiVersion, tt.paramType.GetAPIVersion())
				assert.Equal(t, tt.kind, tt.paramType.GetKind())
			}
		})
	}
}
