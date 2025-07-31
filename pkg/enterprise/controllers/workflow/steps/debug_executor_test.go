// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

package steps

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esapi "github.com/external-secrets/external-secrets/apis/enterprise/workflows/v1alpha1"
)

func TestDebugStepExecutor_Execute(t *testing.T) {
	tests := []struct {
		name          string
		step          *esapi.DebugStep
		data          map[string]interface{}
		expectedError error
	}{
		{
			name: "valid message",
			step: &esapi.DebugStep{
				Message: "Hello, {{.Name}}!",
			},
			data:          map[string]interface{}{"Name": "World"},
			expectedError: nil,
		},
		{
			name: "message template missing variable",
			step: &esapi.DebugStep{
				Message: "Hello, {{.MissingVar}}!",
			},
			data:          map[string]interface{}{"Name": "World"},
			expectedError: errors.New("resolving message"),
		},
		{
			name:          "empty message template",
			step:          &esapi.DebugStep{Message: ""},
			data:          map[string]interface{}{},
			expectedError: nil,
		},
		{
			name:          "nil message",
			step:          &esapi.DebugStep{Message: "{{.NilVar}}"},
			data:          nil,
			expectedError: errors.New("resolving message"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &DebugStepExecutor{Step: tt.step}
			client := fake.NewClientBuilder().Build()
			wf := &esapi.Workflow{}
			_, err := executor.Execute(context.Background(), client, wf, tt.data, "test-job")

			if tt.expectedError != nil {
				assert.ErrorContains(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
