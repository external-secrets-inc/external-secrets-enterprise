// generator_test.go
package sendgrid

import (
	"context"
	"errors"
	"testing"

	"github.com/sendgrid/rest"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// MockClient é uma implementação mock da interface Client.
type MockClient struct {
	GetAPIResponse    *rest.Response
	GetAPIError       error
	PostAPIResponse   *rest.Response
	PostAPIError      error
	DeleteAPIResponse *rest.Response
	DeleteAPIError    error
	PutAPIResponse    *rest.Response
	PutAPIError       error
	PatchAPIResponse  *rest.Response
	PatchAPIError     error
	GetRequestErr     error
}

func (m *MockClient) API(request rest.Request) (*rest.Response, error) {
	switch request.Method {
	case rest.Get:
		return m.GetAPIResponse, m.GetAPIError
	case rest.Post:
		return m.PostAPIResponse, m.PostAPIError
	case rest.Delete:
		return m.DeleteAPIResponse, m.DeleteAPIError
	case rest.Patch:
		return m.PatchAPIResponse, m.PatchAPIError
	case rest.Put:
		return m.PutAPIResponse, m.PutAPIError
	default:
		return m.GetAPIResponse, m.GetAPIError
	}
}

func (m *MockClient) GetRequest(apiKey, endpoint, host string) rest.Request {
	return rest.Request{}
}

func (m *MockClient) SetDataResidency(request rest.Request, dataResidency string) (rest.Request, error) {
	return request, nil
}

func TestGenerator_Generate(t *testing.T) {
	kube := fake.NewClientBuilder().WithObjects(&v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sendgrid-api-key",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"apiKey": []byte("foo"),
		},
	}).Build()
	namespace := "default"

	tests := []struct {
		name       string
		jsonSpec   *apiextensions.JSON
		setupMock  func() *MockClient
		expectErr  bool
		expectData map[string][]byte
	}{
		{
			name: "Success",
			jsonSpec: &apiextensions.JSON{
				Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: SendgridAuthorizationToken
metadata:
  name: my-sendgrid-generator
spec:
  scopes:
    - alerts.create
    - alerts.read
  auth: 
    secretRef:
      apiKeySecretRef:
        name: sendgrid-api-key
        key: apiKey
`),
			},
			setupMock: func() *MockClient {
				return &MockClient{
					GetAPIResponse: &rest.Response{
						StatusCode: 200,
						Body:       `{"result": []}`,
					},
					PostAPIResponse: &rest.Response{
						StatusCode: 200,
						Body:       `{"api_key": "newly-created-api-key"}`,
					},
				}
			},
			expectErr: false,
			expectData: map[string][]byte{
				"apiKey": []byte("newly-created-api-key"),
			},
		},
		{
			name:     "No spec error",
			jsonSpec: nil,
			setupMock: func() *MockClient {
				return &MockClient{}
			},
			expectErr: true,
		}, {
			name: "Creation Failed",
			jsonSpec: &apiextensions.JSON{
				Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: SendgridAuthorizationToken
metadata:
  name: my-sendgrid-generator
spec:
  scopes:
    - alerts.create
    - alerts.read
  auth: 
    secretRef:
      apiKeySecretRef:
        name: sendgrid-api-key
        key: apiKey
`),
			},
			setupMock: func() *MockClient {
				return &MockClient{
					GetAPIResponse: &rest.Response{
						StatusCode: 200,
						Body:       `{"result": []}`,
					},
					PostAPIResponse: &rest.Response{
						StatusCode: 400,
						Body:       `{"error": "bad request"}`,
					},
					PostAPIError: errors.New("an error occurred"),
				}
			},
			expectErr: true,
		}, {
			name: "Deletion Silently Failed",
			jsonSpec: &apiextensions.JSON{
				Raw: []byte(`apiVersion: generators.external-secrets.io/v1alpha1
kind: SendgridAuthorizationToken
metadata:
  name: my-sendgrid-generator
spec:
  scopes:
    - alerts.create
    - alerts.read
  auth: 
    secretRef:
      apiKeySecretRef:
        name: sendgrid-api-key
        key: apiKey
`),
			},
			setupMock: func() *MockClient {
				return &MockClient{
					GetAPIResponse: &rest.Response{
						StatusCode: 200,
						Body:       `{"result": [{"api_key_id": "key-id", "name": "Created By ESO Generator: my-sendgrid-generator"}]}`,
					},
					DeleteAPIResponse: &rest.Response{
						StatusCode: 400,
						Body:       `{"error": "invalid request"}`,
					},
					DeleteAPIError: errors.New("invalid request"),
					PostAPIResponse: &rest.Response{
						StatusCode: 200,
						Body:       `{"api_key": "newly-created-api-key"}`,
					},
				}
			},
			expectErr: false,
			expectData: map[string][]byte{
				"apiKey": []byte("newly-created-api-key"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := &Generator{}

			mockClient := tt.setupMock()
			data, _, err := generator.generate(context.Background(), tt.jsonSpec, kube, namespace, mockClient)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectData, data)
			}
		})
	}
}
