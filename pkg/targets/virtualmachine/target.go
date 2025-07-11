// Copyright External Secrets Inc. 2025
// All Rights reserved.
package virtualmachine

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Provider struct{}

type ScanTarget struct {
	// Virtual Machine Name
	Name              string
	URL               string
	CABundle          []byte
	AuthBasicUsername *string
	AuthBasicPassword *string
	AuthBearerToken   *string
	AuthClientCert    []byte
	AuthClientKey     []byte
	Paths             []string
}

func (p *Provider) NewClient(ctx context.Context, client client.Client, target client.Object) (tgtv1alpha1.ScanTarget, error) {
	converted, ok := target.(*tgtv1alpha1.VirtualMachine)
	if !ok {
		return nil, fmt.Errorf("target %q not found", target.GetObjectKind().GroupVersionKind().Kind)
	}
	var uname, pass string
	var cert, key string
	var err error
	if converted.Spec.Auth != nil {
		if converted.Spec.Auth.Basic != nil {
			uname, err = resolvers.SecretKeyRef(ctx, client, "", converted.GetNamespace(), converted.Spec.Auth.Basic.UsernameSecretRef)
			if err != nil {
				return nil, err
			}
			pass, err = resolvers.SecretKeyRef(ctx, client, "", converted.GetNamespace(), converted.Spec.Auth.Basic.PasswordSecretRef)
			if err != nil {
				return nil, err
			}
		}
		if converted.Spec.Auth.Certificate != nil {
			cert, err = resolvers.SecretKeyRef(ctx, client, "", converted.GetNamespace(), converted.Spec.Auth.Certificate.ClientCertificateSecretRef)
			if err != nil {
				return nil, err
			}
			key, err = resolvers.SecretKeyRef(ctx, client, "", converted.GetNamespace(), converted.Spec.Auth.Certificate.ClientKeySecretRef)
			if err != nil {
				return nil, err
			}
		}
	}
	return &ScanTarget{
		URL:               converted.Spec.URL,
		CABundle:          []byte(converted.Spec.CABundle),
		AuthBasicUsername: &uname,
		AuthBasicPassword: &pass,
		AuthClientCert:    []byte(cert),
		AuthClientKey:     []byte(key),
		Paths:             converted.Spec.Paths,
		Name:              converted.GetName(),
	}, nil
}

func (s *ScanTarget) Scan(ctx context.Context, regexes []string, threshold int) ([]tgtv1alpha1.SecretInStoreRef, error) {
	u, err := url.Parse(s.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL %q: %w", s.URL, err)
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
				return nil, fmt.Errorf("loading client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}
	r := Request{
		Regexes:   regexes,
		Threshold: threshold,
		Paths:     s.Paths,
	}
	body, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	api := fmt.Sprintf("%s/api/v1/scan", s.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, api, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if s.AuthBasicUsername != nil && s.AuthBasicPassword != nil {
		req.SetBasicAuth(*s.AuthBasicUsername, *s.AuthBasicPassword)
	} else if s.AuthBearerToken != nil {
		req.Header.Set("Authorization", "Bearer "+*s.AuthBearerToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	// Parse Response for Job ID;
	var scanResponse ScanResponse
	if err := json.NewDecoder(resp.Body).Decode(&scanResponse); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	// Wait for Job to be completed (timeout of 10 minutes)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	return s.checkForJob(ctx, client, scanResponse.JobId)
}

func (s *ScanTarget) checkForJob(ctx context.Context, client *http.Client, jobID string) ([]tgtv1alpha1.SecretInStoreRef, error) {
	matches, err := s.runMatches(ctx, client, jobID)
	if err != nil && !errors.Is(err, JobNotReadyErr{}) {
		return nil, err
	}
	// If the  first run is ready, lets go forward
	if err == nil {
		return matches, nil
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			matches, err := s.runMatches(ctx, client, jobID)
			if err != nil {
				if errors.Is(err, JobNotReadyErr{}) {
					continue
				}
				return nil, err
			}
			return matches, nil
		}
	}
}

func (s *ScanTarget) runMatches(ctx context.Context, client *http.Client, jobID string) ([]tgtv1alpha1.SecretInStoreRef, error) {
	matches, err := s.getJobMatches(ctx, client, jobID)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func (s *ScanTarget) getJobMatches(ctx context.Context, client *http.Client, jobID string) ([]tgtv1alpha1.SecretInStoreRef, error) {
	scanApi := fmt.Sprintf("%s/api/v1/scan/%s", s.URL, jobID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, scanApi, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.AuthBasicUsername != nil && s.AuthBasicPassword != nil {
		req.SetBasicAuth(*s.AuthBasicUsername, *s.AuthBasicPassword)
	} else if s.AuthBearerToken != nil {
		req.Header.Set("Authorization", "Bearer "+*s.AuthBearerToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var scanJobResponse ScanJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&scanJobResponse); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if scanJobResponse.Status != "completed" {
		return nil, JobNotReadyErr{}
	}
	secrets := []tgtv1alpha1.SecretInStoreRef{}
	for _, match := range scanJobResponse.Match {
		secret := tgtv1alpha1.SecretInStoreRef{
			APIVersion: tgtv1alpha1.SchemeGroupVersion.String(),
			Kind:       tgtv1alpha1.VirtualMachineKind,
			Name:       s.Name,
			RemoteRef: tgtv1alpha1.RemoteRef{
				Key: match.EntryId,
			},
		}
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

type JobNotReadyErr struct{}

func (e JobNotReadyErr) Error() string {
	return "job not ready"
}

func init() {
	tgtv1alpha1.Register(tgtv1alpha1.VirtualMachineKind, &Provider{})
}
