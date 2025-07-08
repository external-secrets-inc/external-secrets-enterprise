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
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

// k8sClient is a global variable that will be set during controller initialization.
var k8sClient client.Client

// SetValidationClient sets the client for validation.
func SetValidationClient(c client.Client) {
	k8sClient = c
}

// validateWorkflowRunParameters validates the arguments in a WorkflowRun against the parameters
// defined in the referenced WorkflowTemplate.
func validateWorkflowRunParameters(wr *WorkflowRun) error {
	if k8sClient == nil {
		return fmt.Errorf("validation client not initialized")
	}

	ctx := context.Background()

	// Fetch the referenced WorkflowTemplate
	template := &WorkflowTemplate{}
	templateNamespace := wr.Namespace
	if wr.Spec.TemplateRef.Namespace != "" {
		templateNamespace = wr.Spec.TemplateRef.Namespace
	}

	if err := k8sClient.Get(ctx, types.NamespacedName{
		Namespace: templateNamespace,
		Name:      wr.Spec.TemplateRef.Name,
	}, template); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("referenced WorkflowTemplate %s not found in namespace %s",
				wr.Spec.TemplateRef.Name, templateNamespace)
		}
		return fmt.Errorf("failed to get referenced WorkflowTemplate: %w", err)
	}

	// Build a map of parameters from the template
	paramMap := make(map[string]*Parameter)
	for _, group := range template.Spec.ParameterGroups {
		for i := range group.Parameters {
			param := &group.Parameters[i]
			paramMap[param.Name] = param
		}
	}

	// Validate each argument against its corresponding parameter
	for argName, argValue := range wr.Spec.Arguments {
		param, exists := paramMap[argName]
		if !exists {
			return fmt.Errorf("argument %q is not defined in the template", argName)
		}

		// Validate the argument value
		if err := validateArgumentValue(ctx, param, argValue, wr.Namespace); err != nil {
			return fmt.Errorf("invalid value for argument %q: %w", argName, err)
		}
	}

	// Check if all required parameters have arguments
	for _, group := range template.Spec.ParameterGroups {
		for _, param := range group.Parameters {
			if param.Required {
				if _, exists := wr.Spec.Arguments[param.Name]; !exists {
					// If a default value is provided, it's okay
					if param.Default == "" {
						return fmt.Errorf("required parameter %q is missing", param.Name)
					}
				}
			}
		}
	}

	return nil
}

// validateArgumentValue validates an argument value against a parameter definition.
func validateArgumentValue(ctx context.Context, param *Parameter, argValue, namespace string) error {
	// For array types (allowMultiple=true), parse as JSON array
	if param.AllowMultiple {
		var arr []interface{}
		if err := json.Unmarshal([]byte(argValue), &arr); err != nil {
			return fmt.Errorf("failed to parse as array: %w", err)
		}

		// Validate each item in the array
		for i, item := range arr {
			if err := validateSingleValue(ctx, param, item, namespace); err != nil {
				return fmt.Errorf("item %d: %w", i, err)
			}
		}

		// Validate array constraints
		if param.Validation != nil {
			if param.Validation.MinItems != nil && len(arr) < *param.Validation.MinItems {
				return fmt.Errorf("requires at least %d items, got %d", *param.Validation.MinItems, len(arr))
			}
			if param.Validation.MaxItems != nil && len(arr) > *param.Validation.MaxItems {
				return fmt.Errorf("allows at most %d items, got %d", *param.Validation.MaxItems, len(arr))
			}
		}
	} else {
		// For primitive types, parse based on the type
		var parsedValue interface{}
		switch param.Type {
		case ParameterTypeNumber:
			var num float64
			if err := json.Unmarshal([]byte(argValue), &num); err != nil {
				return fmt.Errorf("failed to parse as number: %w", err)
			}
			parsedValue = num
		case ParameterTypeBool:
			var b bool
			if err := json.Unmarshal([]byte(argValue), &b); err != nil {
				return fmt.Errorf("failed to parse as boolean: %w", err)
			}
			parsedValue = b
		case ParameterTypeString, ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime,
			ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
			ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
			ParameterTypeGenerator, ParameterTypeGeneratorArray:
			// For string and other types, use the raw value
			parsedValue = argValue
		}

		if param.Type.IsGeneratorType() || param.Type.IsGeneratorArrayType() {
			parsedValue = argValue
		}

		// Validate the single value
		if err := validateSingleValue(ctx, param, parsedValue, namespace); err != nil {
			return err
		}
	}

	return nil
}

// validateSingleValue validates a single value against a parameter definition.
func validateSingleValue(ctx context.Context, param *Parameter, value interface{}, namespace string) error {
	// For multi-select parameters, skip the ValidateValue call since array validation
	// is already handled in validateArgumentValue. We only need to validate individual items.
	if !param.AllowMultiple {
		// First, validate using the existing ValidateValue method for basic type checking
		if err := param.ValidateValue(value); err != nil {
			return err
		}
	} else {
		// For individual items in multi-select parameters, perform basic type validation
		switch param.Type {
		case ParameterTypeNumber:
			_, ok := value.(float64)
			if !ok {
				return fmt.Errorf("must be a number")
			}
		case ParameterTypeBool:
			_, ok := value.(bool)
			if !ok {
				return fmt.Errorf("must be a boolean")
			}
		case ParameterTypeString, ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime,
			ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
			ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
			ParameterTypeGenerator, ParameterTypeGeneratorArray:
			// No specific validation needed for these types
		}
	}

	// For Kubernetes resource types, perform additional validation
	if param.Type.IsKubernetesResource() {
		return validateKubernetesResource(ctx, param, value, namespace)
	}

	return nil
}

// validateKubernetesResource validates that a Kubernetes resource exists and matches constraints.
func validateKubernetesResource(ctx context.Context, param *Parameter, value interface{}, namespace string) error {
	resourceName, ok := value.(string)
	if !ok {
		return fmt.Errorf("kubernetes resource name must be a string. received: %T", value)
	}
	if param.Type == ParameterTypeSecretStoreArray {
		resourceList := strings.Split(resourceName, ",")
		if len(resourceList) > 1 {
			for i := range resourceList {
				if err := validateKubernetesResource(ctx, param, resourceList[i], namespace); err != nil {
					return err
				}
			}
			return nil
		}
	}

	if param.Type.IsGeneratorArrayType() {
		resourceList := strings.Split(resourceName, ",")
		param.Type = ParameterType(fmt.Sprintf("generator[%s]", param.Type.ExtractGeneratorKind()))
		for i := range resourceList {
			if err := validateKubernetesResource(ctx, param, resourceList[i], namespace); err != nil {
				return err
			}
		}
		return nil
	}

	// Determine the resource namespace
	resourceNamespace := namespace
	if param.ResourceConstraints != nil && param.ResourceConstraints.Namespace != "" {
		resourceNamespace = param.ResourceConstraints.Namespace
	}

	// For cluster-scoped resources, don't use a namespace
	if param.Type == ParameterTypeClusterSecretStore {
		resourceNamespace = ""
	}

	// Get the GVK for the resource type
	var gvk schema.GroupVersionKind
	switch param.Type {
	case ParameterTypeSecretStore, ParameterTypeSecretStoreArray, ParameterTypeExternalSecret, ParameterTypeClusterSecretStore:
		gvk = schema.GroupVersionKind{
			Group:   "external-secrets.io",
			Version: "v1",
			Kind:    param.Type.GetKind(),
		}
	case ParameterTypeNamespace:
		gvk = schema.GroupVersionKind{
			Group:   "",
			Version: param.Type.GetAPIVersion(),
			Kind:    param.Type.GetKind(),
		}
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool, ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime:
		// These are not Kubernetes resource types, but we need to handle them for exhaustive switch
		gvk = schema.GroupVersionKind{
			Group:   "",
			Version: param.Type.GetAPIVersion(),
			Kind:    param.Type.GetKind(),
		}
	case ParameterTypeGenerator, ParameterTypeGeneratorArray:
		gvk = schema.GroupVersionKind{
			Group:   "generators.external-secrets.io",
			Version: "v1alpha1",
			Kind:    param.Type.ExtractGeneratorKind(),
		}
	}

	// Special case for Generator type
	if param.Type.IsGeneratorType() {
		gvk = schema.GroupVersionKind{
			Group:   "generators.external-secrets.io",
			Version: "v1alpha1",
			Kind:    param.Type.ExtractGeneratorKind(),
		}
	}

	// Special case for Namespace type
	if param.Type == ParameterTypeNamespace {
		gvk = schema.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Namespace",
		}
		// For namespaces, we don't use the resource namespace
		resourceNamespace = ""
	}

	// Create an unstructured object to fetch the resource
	obj := &unstructured.Unstructured{}
	if !strings.EqualFold(gvk.Kind, "any") {
		obj.SetGroupVersionKind(gvk)

		// Fetch the resource
		err := k8sClient.Get(ctx, types.NamespacedName{
			Namespace: resourceNamespace,
			Name:      resourceName,
		}, obj)

		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Errorf("resource %s of type %s not found in namespace %s",
					resourceName, param.Type, resourceNamespace)
			}
			return fmt.Errorf("error fetching resource: %w", err)
		}
	} else if gvk.Group == "generators.external-secrets.io" { // Handle generator[any]
		generatorFound := false
		scheme := runtime.NewScheme()
		_ = genv1alpha1.AddToScheme(scheme)

		kinds := scheme.KnownTypes(genv1alpha1.SchemeGroupVersion)

		for kind := range kinds {
			// Ignore kubernetes default types and generators lists
			if strings.HasSuffix(kind, "List") ||
				kind == "CreateOptions" ||
				kind == "DeleteOptions" ||
				kind == "GetOptions" ||
				kind == "ListOptions" ||
				kind == "PatchOptions" ||
				kind == "UpdateOptions" ||
				kind == "WatchEvent" {
				continue
			}

			gvk.Kind = kind
			obj.SetGroupVersionKind(gvk)
			err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: resourceNamespace,
				Name:      resourceName,
			}, obj)

			if err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				return fmt.Errorf("error fetching resource: %w", err)
			}

			generatorFound = true
			break
		}

		if !generatorFound {
			return fmt.Errorf("no resource %s of type %s found in namespace %s",
				resourceName, param.Type, resourceNamespace)
		}
	}

	// Check label selector constraints if specified
	if param.ResourceConstraints != nil && len(param.ResourceConstraints.LabelSelector) > 0 {
		labels := obj.GetLabels()
		for key, value := range param.ResourceConstraints.LabelSelector {
			if labels[key] != value {
				return fmt.Errorf("resource does not match label selector: %s=%s", key, value)
			}
		}
	}

	// Check cross-namespace constraints
	if param.ResourceConstraints != nil && !param.ResourceConstraints.AllowCrossNamespace {
		// If the resource is namespace-scoped and we're not allowing cross-namespace references,
		// ensure the resource is in the same namespace as the WorkflowRun
		if resourceNamespace != "" && resourceNamespace != namespace {
			return fmt.Errorf("cross-namespace resource references are not allowed for this parameter")
		}
	}

	return nil
}
