// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.
package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"
)

// PushStepExecutor executes a push step.
type PushStepExecutor struct {
	Step    *workflows.PushStep
	Client  client.Client
	Manager secretstore.ManagerInterface
}

func NewPushStepExecutor(step *workflows.PushStep, c client.Client, manager secretstore.ManagerInterface) *PushStepExecutor {
	if manager == nil {
		panic("manager cannot be nil")
	}

	return &PushStepExecutor{
		Step:    step.DeepCopy(), // Otherwise we are modifying possible templates.
		Client:  c,
		Manager: manager,
	}
}

// Execute pushes secret values to the destination store.
func (e *PushStepExecutor) Execute(ctx context.Context, c client.Client, wf *workflows.Workflow, inputData map[string]interface{}) (map[string]interface{}, error) {
	output := make(map[string]interface{})

	if e.Manager == nil {
		return nil, fmt.Errorf("secret store manager is required")
	}
	defer func() {
		_ = e.Manager.Close(ctx)
	}()

	templates.ProcessTemplates(reflect.ValueOf(e.Step), inputData)

	// Find the source transform data
	secretSource := e.Step.SecretSource
	if secretSource == "" {
		return nil, fmt.Errorf("sourceTransform is required")
	}

	var secret corev1.Secret
	byteData := make(map[string][]byte)

	// For each data item, resolve its value
	for _, data := range e.Step.Data {
		templateStr := fmt.Sprintf("{{ %s.%s }}", secretSource, data.Match.SecretKey)
		value, err := templates.ResolveTemplate(templateStr, inputData)
		if err != nil {
			return nil, fmt.Errorf("error resolving value for key %s: %w", data.Match.SecretKey, err)
		}

		// Check if the value needs to be serialized as JSON
		if v, ok := inputData[secretSource].(map[string]interface{}); ok {
			if fieldValue, exists := v[data.Match.SecretKey]; exists {
				switch val := fieldValue.(type) {
				case nil, string, bool, float64, int, int64:
					// Simple types can be converted to string directly
					byteData[data.Match.SecretKey] = []byte(fmt.Sprintf("%v", val))
				default:
					// Complex types need JSON serialization
					jsonBytes, err := json.Marshal(val)
					if err != nil {
						return nil, fmt.Errorf("error serializing value for key %s: %w", data.Match.SecretKey, err)
					}
					byteData[data.Match.SecretKey] = jsonBytes
				}
				continue
			}
		}
		// Fallback to string value if not a special type
		byteData[data.Match.SecretKey] = []byte(value)
	}

	secret = corev1.Secret{
		Data: byteData,
	}

	for _, data := range e.Step.Data {
		destClient, err := e.Manager.Get(ctx, e.Step.Destination.SecretStoreRef, wf.Namespace, nil)
		if err != nil {
			return nil, fmt.Errorf("error getting destination store client: %w", err)
		}
		err = destClient.PushSecret(ctx, &secret, data)
		if err != nil {
			return nil, fmt.Errorf("error pushing secret data: %w", err)
		}
		output[data.Match.SecretKey] = data.Match.RemoteRef.RemoteKey
	}

	return output, nil
}
