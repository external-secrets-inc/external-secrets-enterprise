// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package steps

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/enterprise/workflows/v1alpha1"
)

// Mock clients.
type mockClient struct {
	client.Client
}

func TestTransformStepExecutor_Execute(t *testing.T) {
	tests := []struct {
		name           string
		step           *esapi.TransformStep
		data           map[string]interface{}
		expectedOutput map[string]interface{}
		expectsError   bool
	}{
		{
			name: "simple mappings",
			step: &esapi.TransformStep{
				Mappings: map[string]string{
					"key1": "{{ .val1 }}",
					"key2": "{{ .val2 }}",
				},
			},
			data: map[string]interface{}{
				"val1": "value1",
				"val2": "value2",
			},
			expectedOutput: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			expectsError: false,
		},
		{
			name: "mapping with invalid template",
			step: &esapi.TransformStep{
				Mappings: map[string]string{
					"invalid": "{{ .missing }}",
				},
			},
			data: map[string]interface{}{
				"val1": "value1",
			},
			expectedOutput: nil,
			expectsError:   true,
		},
		{
			name: "template rendering",
			step: &esapi.TransformStep{
				Template: "Hello, {{ .name }}!",
			},
			data: map[string]interface{}{
				"name": "World",
			},
			expectedOutput: map[string]interface{}{
				"processed": "Hello, World!",
			},
			expectsError: false,
		},
		{
			name: "template with invalid syntax",
			step: &esapi.TransformStep{
				Template: "Hello, {{ .name! }}",
			},
			data: map[string]interface{}{
				"name": "World",
			},
			expectedOutput: nil,
			expectsError:   true,
		},
		{
			name: "combined mappings and template",
			step: &esapi.TransformStep{
				Mappings: map[string]string{
					"key1": "{{ .val1 }}",
				},
				Template: "Greeting: {{ .greeting }}",
			},
			data: map[string]interface{}{
				"val1":     "value1",
				"greeting": "Hello",
			},
			expectedOutput: map[string]interface{}{
				"key1":      "value1",
				"processed": "Greeting: Hello",
			},
			expectsError: false,
		},
		{
			name: "empty step",
			step: &esapi.TransformStep{},
			data: map[string]interface{}{
				"val1": "value1",
			},
			expectedOutput: map[string]interface{}{},
			expectsError:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &TransformStepExecutor{Step: test.step}
			output, err := executor.Execute(context.TODO(), &mockClient{}, &esapi.Workflow{}, test.data, "job-test")

			if test.expectsError {
				assert.NotNil(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, test.expectedOutput, output)
		})
	}
}
