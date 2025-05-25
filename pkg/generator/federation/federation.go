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
package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errNoSpec          = "no config spec provided"
	errParseSpec       = "unable to parse spec: %w"
	errFetchSecretRef  = "could not fetch secret ref: %w"
	errFederationCall  = "failed to call federation server: %w"
	errInvalidResponse = "invalid response from federation server: %w"
)

// Generator implements the generator interface for federation.
type Generator struct{}

// Generate implements the Generator interface.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}

	spec, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	// Get federation server URL
	serverURL := spec.Spec.Server.URL

	// Get auth token
	authToken, err := getFromSecretRef(ctx, spec.Spec.Auth.TokenSecretRef, "", kube, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf(errFetchSecretRef, err)
	}

	// Get CA certificate if provided
	var caCert string
	if spec.Spec.Auth.CACertSecretRef != nil {
		caCert, err = getFromSecretRef(ctx, spec.Spec.Auth.CACertSecretRef, "", kube, namespace)
		if err != nil {
			return nil, nil, fmt.Errorf(errFetchSecretRef, err)
		}
	}

	// Build URL for the federation server's generator endpoint
	url := fmt.Sprintf("%s/generators/%s/%s/%s",
		serverURL,
		spec.Spec.Generator.Namespace,
		spec.Spec.Generator.Kind,
		spec.Spec.Generator.Name)

	// Create payload with CA certificate if provided
	payload := map[string]string{}
	if caCert != "" {
		payload["ca.crt"] = caCert
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	// Send request
	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf(errFederationCall, err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error since we can't return it from a defer
			fmt.Printf("Error closing response body: %v\n", closeErr)
		}
	}()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("federation server returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf(errInvalidResponse, err)
	}

	// Convert string map to byte map
	byteMap := make(map[string][]byte)
	for k, v := range result {
		byteMap[k] = []byte(v)
	}

	return byteMap, nil, nil
}

// Cleanup implements the Generator interface.
func (g *Generator) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, kclient client.Client, namespace string) error {
	// No cleanup needed for federation generator
	return nil
}

// Helper functions.
func parseSpec(data []byte) (*genv1alpha1.Federation, error) {
	var spec genv1alpha1.Federation
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func getFromSecretRef(ctx context.Context, keySelector *esmeta.SecretKeySelector, storeKind string, kube client.Client, namespace string) (string, error) {
	if keySelector == nil {
		return "", errors.New("secret reference is nil")
	}

	value, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, keySelector)
	if err != nil {
		return "", fmt.Errorf(errFetchSecretRef, err)
	}

	return value, err
}

func init() {
	genv1alpha1.Register(string(genv1alpha1.GeneratorKindFederation), &Generator{})
}
