// Copyright External Secrets Inc. All Rights Reserved

package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Generator struct{}

const (
	defaultHost                = "https://api.openai.com/v1"
	organizationPrefix         = "/organization"
	serviceAccountsEndpointFmt = organizationPrefix + "/projects/%s/service_accounts"
	defaultNameSize            = 12
)

type openAiClient struct {
	httpClient *http.Client
	baseURL    string
	authHeader string
	projectID  string
}

func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, kube client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, err
	}

	client, err := newClient(ctx, &res.Spec, kube, namespace)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create OpenAI client: %w", err)
	}

	nameSize := defaultNameSize
	if res.Spec.ServiceAccountNameSize != nil {
		nameSize = *(res.Spec.ServiceAccountNameSize)
	}

	name, err := utils.GenerateRandomString(nameSize)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating random string: %w", err)
	}

	if res.Spec.ServiceAccountNamePrefix != nil {
		name = fmt.Sprintf("%s_%s", *res.Spec.ServiceAccountNamePrefix, name)
	}

	serviceAccount, err := client.createServiceAccount(ctx, name)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create service account: %w", err)
	}

	rawState, err := json.Marshal(&genv1alpha1.OpenAiServiceAccountState{
		ServiceAccountId: serviceAccount.ID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal state: %w", err)
	}

	return map[string][]byte{
		"api_key": []byte(serviceAccount.APIKey.Value),
	}, &apiextensions.JSON{Raw: rawState}, nil
}

func (g *Generator) Cleanup(ctx context.Context, jsonSpec *apiextensions.JSON, previousStatus genv1alpha1.GeneratorProviderState, kclient client.Client, namespace string) error {
	if previousStatus == nil {
		return fmt.Errorf("missing previous status")
	}

	status, err := parseStatus(previousStatus.Raw)
	if err != nil {
		return err
	}

	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return err
	}
	client, err := newClient(ctx, &res.Spec, kclient, namespace)
	if err != nil {
		return err
	}

	err = client.deleteServiceAccount(ctx, status.ServiceAccountId)
	if err != nil {
		return fmt.Errorf("unable to delete service account: %w", err)
	}

	return nil
}

func newClient(ctx context.Context, spec *genv1alpha1.OpenAISpec, kclient client.Client, ns string) (*openAiClient, error) {
	host := defaultHost
	if spec.Host != "" {
		host = spec.Host
	}

	adminApiKey, err := resolvers.SecretKeyRef(ctx, kclient, resolvers.EmptyStoreKind, ns, &esmeta.SecretKeySelector{
		Namespace: &ns,
		Name:      spec.OpenAiAdminKey.Name,
		Key:       spec.OpenAiAdminKey.Key,
	})
	if err != nil {
		return nil, err
	}

	// Prepare the bearer token
	authHeader := fmt.Sprintf("Bearer %s", adminApiKey)

	// Initialize HTTP client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &openAiClient{
		httpClient: httpClient,
		baseURL:    host,
		authHeader: authHeader,
		projectID:  spec.ProjectId,
	}, nil
}

func (c *openAiClient) createServiceAccount(ctx context.Context, name string) (*genv1alpha1.OpenAiServiceAccount, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, fmt.Sprintf(serviceAccountsEndpointFmt, c.projectID))

	payload := map[string]string{
		"name": name,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create service account: %s", resp.Status)
	}

	var result genv1alpha1.OpenAiServiceAccount
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *openAiClient) deleteServiceAccount(ctx context.Context, serviceAccountID string) error {
	url := fmt.Sprintf("%s%s/%s", c.baseURL, fmt.Sprintf(serviceAccountsEndpointFmt, c.projectID), serviceAccountID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete service account: %s", resp.Status)
	}

	return nil
}

func parseSpec(data []byte) (*genv1alpha1.OpenAI, error) {
	var spec genv1alpha1.OpenAI
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func parseStatus(data []byte) (*genv1alpha1.OpenAiServiceAccountState, error) {
	var state genv1alpha1.OpenAiServiceAccountState
	err := json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}
	return &state, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.OpenAiKind, &Generator{})
}
