package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func storeFakeSecret(kube client.Client, namespace, name, key, value string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Data: map[string][]byte{
			key: []byte(value),
		},
	}
	return kube.Create(context.Background(), secret)
}

func TestOpenAiGenerator_GenerateAndCleanup(t *testing.T) {
	mockProjectID := "test-project"

	// Simulate OpenAI service account create response
	serviceAccountResponse := genv1alpha1.OpenAiServiceAccount{
		ID:   "svc_test_123",
		Name: "mock-service-account",
		APIKey: genv1alpha1.OpenAiApiKey{
			Value: "sk-test123",
		},
	}

	// Mock OpenAI Admin API Server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(serviceAccountResponse)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockServer.Close()

	// Create fake Kubernetes client with mocked secret for admin API key
	fakeKube := fake.NewClientBuilder().Build()

	// Store fake admin API key in fake secrets backend
	adminKey := "fake-admin-key"
	err := storeFakeSecret(fakeKube, "default", "openai-admin-key", "api-key", adminKey)
	require.NoError(t, err)

	// Prepare generator spec
	spec := genv1alpha1.OpenAI{
		Spec: genv1alpha1.OpenAISpec{
			Host:      mockServer.URL, // override to mock server
			ProjectId: mockProjectID,
			OpenAiAdminKey: genv1alpha1.SecretKeySelector{
				Name: "openai-admin-key",
				Key:  "api-key",
			},
		},
	}

	specRaw, _ := json.Marshal(spec)

	// Initialize generator
	gen := &Generator{}

	// Call Generate()
	secrets, state, err := gen.Generate(context.Background(), &apiextensions.JSON{Raw: specRaw}, fakeKube, "default")
	require.NoError(t, err)
	require.NotNil(t, secrets)
	require.NotEmpty(t, state)

	assert.Contains(t, secrets, "id")
	assert.Contains(t, secrets, "api_key")
	assert.Equal(t, "svc_test_123", string(secrets["id"]))
	assert.Equal(t, "sk-test123", string(secrets["api_key"]))

	// Call Cleanup()
	err = gen.Cleanup(context.Background(), &apiextensions.JSON{Raw: specRaw}, state, fakeKube, "default")
	require.NoError(t, err)
}
