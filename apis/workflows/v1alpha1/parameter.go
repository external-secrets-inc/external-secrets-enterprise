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
package v1alpha1

import (
	"fmt"
)

// IsPrimitive returns true if the parameter type is a primitive value.
func (p ParameterType) IsPrimitive() bool {
	switch p {
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime:
		return true
	case ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
		ParameterTypeClusterSecretStore, ParameterTypeGenerator:
		return false
	default:
		return false
	}
}

// IsKubernetesResource returns true if the parameter type represents a Kubernetes resource.
func (p ParameterType) IsKubernetesResource() bool {
	switch p {
	case ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
		ParameterTypeClusterSecretStore, ParameterTypeGenerator:
		return true
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime:
		return false
	default:
		return false
	}
}

// GetAPIVersion returns the API version for Kubernetes resource types.
func (p ParameterType) GetAPIVersion() string {
	switch p {
	case ParameterTypeNamespace:
		return "v1"
	case ParameterTypeSecretStore, ParameterTypeExternalSecret:
		return "external-secrets.io/v1"
	case ParameterTypeClusterSecretStore:
		return "external-secrets.io/v1"
	case ParameterTypeGenerator:
		return "v1alpha1"
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime:
		return ""
	default:
		return ""
	}
}

// GetKind returns the Kind for Kubernetes resource types.
func (p ParameterType) GetKind() string {
	switch p {
	case ParameterTypeNamespace:
		return "Namespace"
	case ParameterTypeSecretStore:
		return "SecretStore"
	case ParameterTypeExternalSecret:
		return "ExternalSecret"
	case ParameterTypeClusterSecretStore:
		return "ClusterSecretStore"
	case ParameterTypeGenerator:
		return "Generator"
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime:
		return ""
	default:
		return ""
	}
}

// IsMultiSelect returns true if the parameter allows multiple selections.
func (p *Parameter) IsMultiSelect() bool {
	return p.AllowMultiple
}

// GetExpectedFormat returns the expected format for the parameter value.
func (p *Parameter) GetExpectedFormat() string {
	if p.IsMultiSelect() {
		return "array"
	}
	return string(p.Type)
}

// ValidateValue validates a parameter value against its constraints.
func (p *Parameter) ValidateValue(value interface{}) error {
	if p.IsMultiSelect() {
		// Expect an array for multi-select parameters
		arr, ok := value.([]interface{})
		if !ok {
			// Also check for string-encoded array (e.g., from JSON)
			strVal, isStr := value.(string)
			if isStr && (strVal != "" && strVal[0] == '[' && strVal[len(strVal)-1] == ']') {
				// This appears to be a JSON array string, which will be parsed later
				return nil
			}
			return fmt.Errorf("expected array for multi-select parameter %s", p.Name)
		}

		if p.Validation != nil {
			if p.Validation.MinItems != nil && len(arr) < *p.Validation.MinItems {
				return fmt.Errorf("parameter %s requires at least %d items", p.Name, *p.Validation.MinItems)
			}
			if p.Validation.MaxItems != nil && len(arr) > *p.Validation.MaxItems {
				return fmt.Errorf("parameter %s allows at most %d items", p.Name, *p.Validation.MaxItems)
			}
		}

		// Type-specific validation for array elements
		switch p.Type {
		case ParameterTypeNumber:
			for i, item := range arr {
				_, ok := item.(float64)
				if !ok {
					return fmt.Errorf("item %d in parameter %s must be a number", i, p.Name)
				}
			}
		case ParameterTypeBool:
			for i, item := range arr {
				_, ok := item.(bool)
				if !ok {
					return fmt.Errorf("item %d in parameter %s must be a boolean", i, p.Name)
				}
			}
		case ParameterTypeString, ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime,
			ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
			ParameterTypeClusterSecretStore, ParameterTypeGenerator:
			// No specific validation needed for these types in array context
		}
	} else {
		// Type-specific validation for single values
		switch p.Type {
		case ParameterTypeNumber:
			_, ok := value.(float64)
			if !ok {
				return fmt.Errorf("parameter %s must be a number", p.Name)
			}
		case ParameterTypeBool:
			_, ok := value.(bool)
			if !ok {
				return fmt.Errorf("parameter %s must be a boolean", p.Name)
			}
		case ParameterTypeString, ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime,
			ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
			ParameterTypeClusterSecretStore, ParameterTypeGenerator:
			// No specific validation needed for these types in single value context
		}
	}
	return nil
}
