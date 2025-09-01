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
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/targets"
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
	var newVal []byte
	var ok bool
	if remoteRef.GetSecretKey() == "" {
		// Get The full Secret
		d, err := json.Marshal(secret.Data)
		if err != nil {
			return fmt.Errorf("error marshaling secret: %w", err)
		}
		newVal = d
	} else {
		newVal, ok = secret.Data[remoteRef.GetSecretKey()]
		if !ok {
			return fmt.Errorf("secret key %q not found", remoteRef.GetSecretKey())
		}
	}

	client := &http.Client{}

	if u.Scheme == HTTPS {
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
		Value: string(newVal),
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

	err = targets.UpdateTargetPushIndex(ctx, s.KubeClient, s.Name, s.Namespace, remoteRef.GetRemoteKey(), remoteRef.GetProperty(), targets.Hash(newVal))
	if err != nil {
		return fmt.Errorf("error updating target status: %w", err)
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
	if s.URL == "" {
		return esv1.ValidationResultError, fmt.Errorf("error: missing URL")
	}
	u, err := url.Parse(s.URL)
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("error parsing URL %q: %w", s.URL, err)
	}

	client := &http.Client{}
	if u.Scheme == HTTPS {
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
		if len(s.CABundle) > 0 {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(s.CABundle)
			tlsConfig.RootCAs = caCertPool
		}
		if len(s.AuthClientCert) > 0 && len(s.AuthClientKey) > 0 {
			cert, err := tls.X509KeyPair(s.AuthClientCert, s.AuthClientKey)
			if err != nil {
				return esv1.ValidationResultError, fmt.Errorf("error loading client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		client.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	}

	// Minimal, harmless scan payload just to validate auth/connectivity.
	// We do NOT wait for the job to complete.
	payload := Request{
		Regexes:   []string{"__eso_validate__"},
		Threshold: 0,
		Paths:     s.Paths,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("error marshaling validation payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	api := fmt.Sprintf("%s/api/v1/scan", s.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api, bytes.NewReader(body))
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("error creating validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.AuthBasicUsername != nil && s.AuthBasicPassword != nil {
		req.SetBasicAuth(*s.AuthBasicUsername, *s.AuthBasicPassword)
	} else if s.AuthBearerToken != nil {
		req.Header.Set("Authorization", "Bearer "+*s.AuthBearerToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return esv1.ValidationResultError, fmt.Errorf("validation request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return esv1.ValidationResultError, fmt.Errorf("unauthorized to access %s: http %d", api, resp.StatusCode)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return esv1.ValidationResultError, fmt.Errorf("error at validation request: http %d", resp.StatusCode)
	}

	return esv1.ValidationResultReady, nil
}
