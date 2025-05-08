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
package sendgrid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/sendgrid/rest"
	sendgridapi "github.com/sendgrid/sendgrid-go"
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
	errDeleteAPIKey    = "failed to delete existing API key: %w"
	errCreateAPIKey    = "failed to create new API key: %w"
	errGetAPIKeys      = "failed to get API keys: %w"
	errBuildPayload    = "failed to build payload: %w"
	errProcessResponse = "failed to process response: %w"
	errBuildRequest    = "failed to build SendGrid request: %w"
)

type SecretKey struct {
	ID     string   `json:"api_key_id,omitempty"`
	Key    string   `json:"api_key,omitempty"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes,omitempty"`
}

type SecretKeyList struct {
	Keys []SecretKey `json:"result"`
}

func (l *SecretKeyList) filterByName(name string) []SecretKey {
	var filtered []SecretKey
	for _, key := range l.Keys {
		if key.Name == name {
			filtered = append(filtered, key)
		}
	}
	return filtered
}

type Client interface {
	API(request rest.Request) (*rest.Response, error)
	GetRequest(apiKey, endpoint, host string) rest.Request
	SetDataResidency(request rest.Request, dataResidency string) (rest.Request, error)
}

type SendGridClient struct{}

func (c *SendGridClient) API(request rest.Request) (*rest.Response, error) {
	return sendgridapi.API(request)
}

func (c *SendGridClient) GetRequest(apiKey, endpoint, host string) rest.Request {
	return sendgridapi.GetRequest(apiKey, endpoint, host)
}

func (c *SendGridClient) SetDataResidency(request rest.Request, dataResidency string) (rest.Request, error) {
	return sendgridapi.SetDataResidency(request, dataResidency)
}

type Generator struct{}

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	client := &SendGridClient{}
	return g.generate(ctx, jsonSpec, kube, namespace, client)
}

func (g *Generator) generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string, client Client) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	secretName := fmt.Sprintf("Created By ESO Generator: %s", res.ObjectMeta.Name)

	dataResidency := res.Spec.DataResidency
	apiKey, err := getFromSecretRef(ctx, &res.Spec.Auth.SecretRef.APIKey, "", kube, namespace)
	if err != nil {
		return nil, nil, err
	}

	secretKeys, err := g.getExistingAPIKeys(apiKey, dataResidency, client)
	if err != nil {
		return nil, nil, err
	}

	keysToDelete := secretKeys.filterByName(secretName)
	if err := g.deleteAPIKeys(keysToDelete, apiKey, dataResidency, client); err != nil {
		return nil, nil, err
	}

	createdSecret, err := g.createAPIKey(secretName, res.Spec.Scopes, apiKey, dataResidency, client)
	if err != nil {
		return nil, nil, err
	}

	return map[string][]byte{
		"apiKey": []byte(createdSecret.Key),
	}, nil, nil
}

func (g *Generator) buildSendGridRequest(apiKey, dataResidency string, method rest.Method, endpoint string, client Client) (rest.Request, error) {
	request := client.GetRequest(apiKey, endpoint, "")
	request.Method = method
	request, err := client.SetDataResidency(request, dataResidency)
	if err != nil {
		return request, fmt.Errorf(errBuildRequest, err)
	}
	return request, nil
}

func (g *Generator) getExistingAPIKeys(apiKey, dataResidency string, client Client) (SecretKeyList, error) {
	getAPIKeysRequest, err := g.buildSendGridRequest(apiKey, dataResidency, rest.Get, "/v3/api_keys", client)
	if err != nil {
		return SecretKeyList{}, fmt.Errorf(errBuildRequest, err)
	}

	response, err := client.API(getAPIKeysRequest)
	if err != nil {
		return SecretKeyList{}, fmt.Errorf(errGetAPIKeys, err)
	}

	var secretKeys SecretKeyList
	if err := json.Unmarshal([]byte(response.Body), &secretKeys); err != nil {
		return SecretKeyList{}, fmt.Errorf(errProcessResponse, err)
	}

	return secretKeys, nil
}

func (g *Generator) deleteAPIKeys(apiKeys []SecretKey, apiKey, dataResidency string, client Client) error {
	for _, key := range apiKeys {
		path := fmt.Sprintf("/v3/api_keys/%s", key.ID)
		deleteRequest, err := g.buildSendGridRequest(apiKey, dataResidency, rest.Delete, path, client)
		if err != nil {
			return fmt.Errorf(errBuildRequest, err)
		}
		if _, err := client.API(deleteRequest); err != nil {
			// Silently ignore errors when deleting old API Keys because it doesn't prevent the creation of a new API Key.
			// Old API Keys will be retried for deletion in the next secret generation.
			log.Printf("failed to delete API Key with ID %s: %v", key.ID, err)
			continue
		}
	}
	return nil
}

func (g *Generator) createAPIKey(secretName string, scopes []string, apiKey, dataResidency string, client Client) (SecretKey, error) {
	createAPIKeyRequest, err := g.buildSendGridRequest(apiKey, dataResidency, rest.Post, "/v3/api_keys", client)
	if err != nil {
		return SecretKey{}, fmt.Errorf(errBuildRequest, err)
	}

	apiKeyData := SecretKey{
		Name:   secretName,
		Scopes: scopes,
	}
	body, err := json.Marshal(apiKeyData)
	if err != nil {
		return SecretKey{}, fmt.Errorf(errBuildPayload, err)
	}
	createAPIKeyRequest.Body = body

	response, err := client.API(createAPIKeyRequest)
	if err != nil {
		return SecretKey{}, fmt.Errorf(errCreateAPIKey, err)
	}

	var createdSecret SecretKey
	if err := json.Unmarshal([]byte(response.Body), &createdSecret); err != nil {
		return SecretKey{}, fmt.Errorf(errProcessResponse, err)
	}

	return createdSecret, nil
}

func parseSpec(data []byte) (*genv1alpha1.SendgridAuthorizationToken, error) {
	var spec genv1alpha1.SendgridAuthorizationToken

	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func getFromSecretRef(ctx context.Context, keySelector *esmeta.SecretKeySelector, storeKind string, kube client.Client, namespace string) (string, error) {
	value, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, keySelector)
	if err != nil {
		return "", fmt.Errorf(errFetchSecretRef, err)
	}

	return value, err
}

func (g *Generator) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, state genv1alpha1.GeneratorProviderState, kclient client.Client, namespace string) error {
	return nil
}

func init() {
	genv1alpha1.Register(genv1alpha1.SendgridKind, &Generator{})
}
