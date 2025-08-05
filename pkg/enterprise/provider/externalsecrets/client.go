// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
package externalsecrets

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errNotImplemented = "not implemented"
)

type Client struct {
	httpClient      *http.Client
	kclient         kclient.Client
	serverURL       string
	localCaRef      []byte
	token           string
	secretStoreName string
}

// TODO - implement client side and server side.
func (g *Client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

// TODO - implement client side and server side.
func (g *Client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// TODO - implement client side and server side.
func (g *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

// TODO - implement client side and server side.
func (g *Client) GetAllSecrets(_ context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// TODO - implement client side and server side.
func (g *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

func (g *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	realURL := fmt.Sprintf("%s/secretstore/%s/secrets/%s", g.serverURL, g.secretStoreName, ref.Key)
	serverURL, err := url.Parse(realURL)
	if err != nil {
		return nil, err
	}
	headers := http.Header{}
	headers.Add("Authorization", fmt.Sprintf("Bearer %s", g.token))
	headers.Add("Content-Type", "application/json")
	b64cert := base64.StdEncoding.EncodeToString(g.localCaRef)
	preview := `{"ca.crt": "%s"}`
	body := fmt.Sprintf(preview, b64cert)
	req := http.Request{
		Method: "POST",
		URL:    serverURL,
		Header: headers,
		Body:   io.NopCloser(bytes.NewReader([]byte(body))),
	}

	res, err := g.httpClient.Do(&req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("failed to get secret: status %s, body: %s", res.Status, resBody)
	}
	return resBody, nil
}

func (g *Client) Close(_ context.Context) error {
	return nil
}

func (g *Client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}
