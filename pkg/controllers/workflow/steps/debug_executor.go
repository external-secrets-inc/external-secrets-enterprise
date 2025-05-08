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
package steps

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"
)

type DebugStepExecutor struct {
	Step *esapi.DebugStep
}

func NewDebugStepExecutor(step *esapi.DebugStep) *DebugStepExecutor {
	return &DebugStepExecutor{
		Step: step,
	}
}

func (e *DebugStepExecutor) Execute(ctx context.Context, client client.Client, wf *esapi.Workflow, data map[string]interface{}) (map[string]interface{}, error) {
	message, err := templates.ResolveTemplate(e.Step.Message, data)
	if err != nil {
		return nil, fmt.Errorf("resolving message: %w", err)
	}
	fmt.Println("Debug message:", message) // Replace with your logger

	// Return the message as an output
	return map[string]interface{}{
		"message": message,
	}, nil
}
