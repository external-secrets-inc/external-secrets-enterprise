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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// TransformStepExecutor handles the execution of transform steps in a workflow.
type TransformStepExecutor struct {
	Step *workflows.TransformStep
}

func NewTransformStepExecutor(step *workflows.TransformStep) *TransformStepExecutor {
	return &TransformStepExecutor{
		Step: step,
	}
}

// Execute processes the transform step using templatev2 engine.
func (e *TransformStepExecutor) Execute(ctx context.Context, client client.Client, wf *workflows.Workflow, data map[string]interface{}) (map[string]interface{}, error) {
	outputs := make(map[string]interface{})

	// Create template engine with sprig functions
	tmplEngine := template.New("transform").
		Funcs(sprig.TxtFuncMap()). // Add sprig functions
		Option("missingkey=error") // Fail on missing keys

	// If full template is provided, process it
	if e.Step.Template != "" {
		tmpl, err := tmplEngine.Parse(e.Step.Template)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to execute template: %w", err)
		}

		// Try to parse as JSON first
		var jsonValue interface{}
		if err := json.Unmarshal(buf.Bytes(), &jsonValue); err == nil {
			outputs["processed"] = jsonValue
		} else {
			// If not valid JSON, store as string
			outputs["processed"] = buf.String()
		}
	}

	// Process individual mappings
	for key, templateStr := range e.Step.Mappings {
		tmpl, err := tmplEngine.Parse(templateStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse mapping template for key %s: %w", key, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to execute mapping template for key %s: %w", key, err)
		}

		// Try to parse as JSON first
		var jsonValue interface{}
		if err := json.Unmarshal(buf.Bytes(), &jsonValue); err == nil {
			outputs[key] = jsonValue
		} else {
			// If not valid JSON, store as string
			outputs[key] = buf.String()
		}
	}

	return outputs, nil
}
