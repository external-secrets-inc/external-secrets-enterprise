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
	"encoding/json"
	"fmt"
	"regexp"
)

var generatorPattern = regexp.MustCompile(string(ParameterTypeGenerator))
var generatorArrayPattern = regexp.MustCompile(string(ParameterTypeGeneratorArray))

// IsGeneratorType checks if the value matches the pattern generator[kind].
func (p ParameterType) IsGeneratorType() bool {
	return generatorPattern.MatchString(string(p))
}

// IsGeneratorType checks if the value matches the pattern array[generator[kind]].
func (p ParameterType) IsGeneratorArrayType() bool {
	return generatorArrayPattern.MatchString(string(p))
}

// ExtractGeneratorKind returns the kind inside generator[kind] or array[generator[kind]], or empty string if invalid.
func (p ParameterType) ExtractGeneratorKind() string {
	str := string(p)

	// Try direct generator[kind]
	if matches := generatorPattern.FindStringSubmatch(str); len(matches) == 2 {
		return matches[1]
	}

	// Try array[generator[kind]]
	if matches := generatorArrayPattern.FindStringSubmatch(str); len(matches) == 2 {
		return matches[1]
	}

	return ""
}

// IsPrimitive returns true if the parameter type is a primitive value.
func (p ParameterType) IsPrimitive() bool {
	if p.IsGeneratorType() || p.IsGeneratorArrayType() {
		return false
	}

	switch p {
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime:
		return true
	case ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
		ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
		ParameterTypeGenerator, ParameterTypeGeneratorArray,
		ParameterTypeSecretLocation, ParameterTypeSecretLocationArray,
		ParameterTypeFinding, ParameterTypeFindingArray:
		return false
	default:
		return false
	}
}

// IsKubernetesResource returns true if the parameter type represents a Kubernetes resource.
func (p ParameterType) IsKubernetesResource() bool {
	if p.IsGeneratorType() || p.IsGeneratorArrayType() {
		return true
	}

	switch p {
	case ParameterTypeNamespace, ParameterTypeSecretStore, ParameterTypeExternalSecret,
		ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
		ParameterTypeGenerator, ParameterTypeGeneratorArray,
		ParameterTypeSecretLocation, ParameterTypeSecretLocationArray,
		ParameterTypeFinding, ParameterTypeFindingArray:
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
	if p.IsGeneratorType() || p.IsGeneratorArrayType() {
		return "v1alpha1"
	}

	switch p {
	case ParameterTypeNamespace:
		return "v1"
	case ParameterTypeSecretStore, ParameterTypeExternalSecret, ParameterTypeSecretStoreArray,
		ParameterTypeSecretLocation, ParameterTypeSecretLocationArray:
		return "external-secrets.io/v1"
	case ParameterTypeClusterSecretStore:
		return "external-secrets.io/v1"
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime:
		return ""
	case ParameterTypeGenerator, ParameterTypeGeneratorArray:
		return "v1alpha1"
	case ParameterTypeFinding, ParameterTypeFindingArray:
		return "scan.external-secrets.io/v1alpha1"
	default:
		return ""
	}
}

// GetKind returns the Kind for Kubernetes resource types.
func (p ParameterType) GetKind() string {
	if p.IsGeneratorType() || p.IsGeneratorArrayType() {
		return p.ExtractGeneratorKind()
	}

	switch p {
	case ParameterTypeNamespace:
		return "Namespace"
	case ParameterTypeSecretStore, ParameterTypeSecretStoreArray,
		ParameterTypeSecretLocation, ParameterTypeSecretLocationArray:
		return "SecretStore"
	case ParameterTypeExternalSecret:
		return "ExternalSecret"
	case ParameterTypeClusterSecretStore:
		return "ClusterSecretStore"
	case ParameterTypeString, ParameterTypeNumber, ParameterTypeBool,
		ParameterTypeObject, ParameterTypeSecret, ParameterTypeTime:
		return ""
	case ParameterTypeGenerator, ParameterTypeGeneratorArray:
		return p.ExtractGeneratorKind()
	case ParameterTypeFinding, ParameterTypeFindingArray:
		return "Finding"
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
			ParameterTypeNamespace, ParameterTypeExternalSecret:
			for i, item := range arr {
				_, ok := item.(string)
				if !ok {
					return fmt.Errorf("item %d in parameter %s must be a string", i, p.Name)
				}
			}
		case ParameterTypeSecretStore, ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
			ParameterTypeGenerator, ParameterTypeGeneratorArray,
			ParameterTypeSecretLocation, ParameterTypeSecretLocationArray,
			ParameterTypeFinding, ParameterTypeFindingArray:

			converters := p.GetConverters()
			converter := converters[p.Type]

			for i, item := range arr {
				_, err := converter(item)
				if err != nil {
					return fmt.Errorf("item %d error: %w", i, err)
				}
			}
		}

		if p.Type.IsGeneratorType() {
			for i, item := range arr {
				_, err := p.ToGeneratorParameterType(item)
				if err != nil {
					return fmt.Errorf("item %d error: %w", i, err)
				}
			}
		}

		if p.Type.IsGeneratorArrayType() {
			for i, item := range arr {
				_, err := p.ToGeneratorParameterTypeArray(item)
				if err != nil {
					return fmt.Errorf("item %d error: %w", i, err)
				}
			}
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
			ParameterTypeNamespace, ParameterTypeExternalSecret:
			_, ok := value.(string)
			if !ok {
				return fmt.Errorf("parameter %s must be a string", p.Name)
			}
		case ParameterTypeSecretStore, ParameterTypeClusterSecretStore, ParameterTypeSecretStoreArray,
			ParameterTypeGenerator, ParameterTypeGeneratorArray,
			ParameterTypeSecretLocation, ParameterTypeSecretLocationArray,
			ParameterTypeFinding, ParameterTypeFindingArray:

			converters := p.GetConverters()
			converter := converters[p.Type]
			_, err := converter(value)
			if err != nil {
				return err
			}
		}

		if p.Type.IsGeneratorType() {
			_, err := p.ToGeneratorParameterType(value)
			if err != nil {
				return err
			}
		}

		if p.Type.IsGeneratorArrayType() {
			_, err := p.ToGeneratorParameterTypeArray(value)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p Parameter) ToSecretStoreParameterType(value interface{}) (*SecretStoreParameterType, error) {
	var resource SecretStoreParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format {\"name\": \"store-name\"}. received: %T", p.Type, value)
	}
	return &resource, nil
}

func (p Parameter) ToGeneratorParameterType(value interface{}) (*GeneratorParameterType, error) {
	var resource GeneratorParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format {\"name\": \"store-name\", \"kind\":\"Kind\"}. received: %T", p.Type, value)
	}

	if resource.Name == nil || resource.Kind == nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format {\"name\": \"store-name\", \"kind\":\"Kind\"}. received: %T", p.Type, value)
	}

	return &resource, nil
}

func (p Parameter) ToSecretLocationParameterType(value interface{}) (*SecretLocationParameterType, error) {
	var resource SecretLocationParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %s must be an object of the format {\"name\": \"store-name\", \"apiVersion\": \"v1\", \"kind\": \"Kind\", \"remoteRef\": {\"key\": \"remote-key\", \"property\": \"remote-property\"}}. received: %T",
			p.Type, value,
		)
	}

	if resource.Name == "" ||
		resource.APIVersion == "" ||
		resource.Kind == "" ||
		resource.RemoteRef.Key == "" {
		return nil, fmt.Errorf(
			"parameter %s must be an object of the format {\"name\": \"store-name\", \"apiVersion\": \"v1\", \"kind\": \"Kind\", \"remoteRef\": {\"key\": \"remote-key\"}}. received: %T",
			p.Type, value,
		)
	}

	return &resource, nil
}

func (p Parameter) ToFindingParameterType(value interface{}) (*FindingParameterType, error) {
	var resource FindingParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format {\"name\": \"finding-name\"}. received: %T", p.Type, value)
	}
	return &resource, nil
}

func (p Parameter) ToSecretStoreParameterTypeArray(value interface{}) ([]SecretStoreParameterType, error) {
	var resource []SecretStoreParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format [{\"name\": \"store-name\"}]. received: %T", p.Type, value)
	}
	return resource, nil
}

func (p Parameter) ToGeneratorParameterTypeArray(value interface{}) ([]GeneratorParameterType, error) {
	var resource []GeneratorParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format [{\"name\": \"store-name\", \"kind\":\"Kind\"}]. received: %T", p.Type, value)
	}
	return resource, nil
}

func (p Parameter) ToSecretLocationParameterTypeArray(value interface{}) ([]SecretLocationParameterType, error) {
	var resource []SecretLocationParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %s must be an object of the format [{\"name\": \"store-name\", \"apiVersion\": \"v1\", \"kind\": \"Kind\", \"remoteRef\": {\"key\": \"remote-key\", \"property\": \"remote-property\"}}]. received: %T",
			p.Type, value,
		)
	}
	return resource, nil
}

func (p Parameter) ToFindingParameterTypeArray(value interface{}) ([]FindingParameterType, error) {
	var resource []FindingParameterType
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("error marshaling parameter %s. received: %T", p.Name, value)
	}

	err = json.Unmarshal(valueBytes, &resource)
	if err != nil {
		return nil, fmt.Errorf("parameter %s must be an object of the format [{\"name\": \"finding-name\"}]. received: %T", p.Type, value)
	}
	return resource, nil
}

type converterFunc func(value interface{}) (any, error)

func wrapConverter[T any](fn func(value interface{}) (*T, error)) converterFunc {
	return func(value interface{}) (any, error) {
		return fn(value)
	}
}

func wrapConverterArray[T any](fn func(value interface{}) ([]T, error)) converterFunc {
	return func(value interface{}) (any, error) {
		return fn(value)
	}
}

func (p Parameter) GetConverters() map[ParameterType]converterFunc {
	return map[ParameterType]converterFunc{
		ParameterTypeSecretStore:         wrapConverter(p.ToSecretStoreParameterType),
		ParameterTypeClusterSecretStore:  wrapConverter(p.ToSecretStoreParameterType),
		ParameterTypeSecretStoreArray:    wrapConverterArray(p.ToSecretStoreParameterTypeArray),
		ParameterTypeGenerator:           wrapConverter(p.ToGeneratorParameterType),
		ParameterTypeGeneratorArray:      wrapConverterArray(p.ToGeneratorParameterTypeArray),
		ParameterTypeSecretLocation:      wrapConverter(p.ToSecretLocationParameterType),
		ParameterTypeSecretLocationArray: wrapConverterArray(p.ToSecretLocationParameterTypeArray),
		ParameterTypeFinding:             wrapConverter(p.ToFindingParameterType),
		ParameterTypeFindingArray:        wrapConverterArray(p.ToFindingParameterTypeArray),
	}
}
