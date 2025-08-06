// Copyright External Secrets Inc. 2025
// All rights reserved.
package virtualmachine

import (
	"bytes"
	"context"
	"crypto/sha3"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	corev1 "k8s.io/api/core/v1"
)

func (s *ScanTarget) PushSecret(ctx context.Context, secret *corev1.Secret, remoteRef esv1.PushSecretData) error {
	u, err := url.Parse(s.URL)
	if err != nil {
		return fmt.Errorf("parsing URL %q: %w", s.URL, err)
	}
	if remoteRef.GetProperty() == "" {
		return errors.New(errPropertyMandatory)
	}
	var value []byte
	var ok bool
	if remoteRef.GetSecretKey() == "" {
		// Get The full Secret
		d, err := json.Marshal(secret.Data)
		if err != nil {
			return fmt.Errorf("error marshaling secret: %w", err)
		}
		value = d
	} else {
		value, ok = secret.Data[remoteRef.GetSecretKey()]
		if !ok {
			return fmt.Errorf("secret key %q not found", remoteRef.GetSecretKey())
		}
	}

	client := &http.Client{}

	if u.Scheme == "https" {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		if len(s.CABundle) > 0 {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(s.CABundle)
			tlsConfig.RootCAs = caCertPool
		}

		if len(s.AuthClientCert) > 0 && len(s.AuthClientKey) > 0 {
			cert, err := tls.X509KeyPair(s.AuthClientCert, s.AuthClientKey)
			if err != nil {
				return fmt.Errorf("loading client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}
	r := PushRequest{
		Value: string(value),
	}
	body, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}
	idx := fmt.Sprintf("%v@%v", remoteRef.GetRemoteKey(), remoteRef.GetProperty())
	fingerprint := sha3.New224().Sum([]byte(idx))
	api := fmt.Sprintf("%s/api/v1/secrets/%x/version", s.URL, fingerprint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if s.AuthBasicUsername != nil && s.AuthBasicPassword != nil {
		req.SetBasicAuth(*s.AuthBasicUsername, *s.AuthBasicPassword)
	} else if s.AuthBearerToken != nil {
		req.Header.Set("Authorization", "Bearer "+*s.AuthBearerToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (s *ScanTarget) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

func (s *ScanTarget) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

func (s *ScanTarget) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented - this provider supports write-only operations")
}

func (s *ScanTarget) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, fmt.Errorf("not implemented - this provider supports write-only operations")
}

func (s *ScanTarget) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented - this provider supports write-only operations")
}

func (s *ScanTarget) Close(ctx context.Context) error {
	ctx.Done()
	return nil
}

func (s *ScanTarget) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultUnknown, nil
}
