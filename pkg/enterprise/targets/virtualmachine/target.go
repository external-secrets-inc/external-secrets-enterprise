// Copyright External Secrets Inc. 2025
// All Rights reserved.
package virtualmachine

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var mu sync.Mutex

type Provider struct{}

type ScanTarget struct {
	// Virtual Machine Name
	Name              string
	Namespace         string
	URL               string
	CABundle          []byte
	AuthBasicUsername *string
	AuthBasicPassword *string
	AuthBearerToken   *string
	AuthClientCert    []byte
	AuthClientKey     []byte
	Paths             []string
	KubeClient        client.Client
}

const (
	errNotImplemented    = "not implemented"
	errPropertyMandatory = "property is mandatory"
	HTTPS                = "https"
)

func (p *Provider) NewClient(ctx context.Context, client client.Client, target client.Object) (tgtv1alpha1.ScanTarget, error) {
	converted, ok := target.(*tgtv1alpha1.VirtualMachine)
	if !ok {
		return nil, fmt.Errorf("target %q not found", target.GetObjectKind().GroupVersionKind().Kind)
	}
	uname, pass, err := getBasicAuth(ctx, client, converted.GetNamespace(), converted.Spec.Auth)
	if err != nil {
		return nil, err
	}
	cert, key, err := getCertAuth(ctx, client, converted.GetNamespace(), converted.Spec.Auth)
	if err != nil {
		return nil, err
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
		Namespace:         converted.GetNamespace(),
		KubeClient:        client,
	}, nil
}

type SecretStoreProvider struct {
}

func (p *SecretStoreProvider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreWriteOnly
}

func (p *SecretStoreProvider) ValidateStore(_ esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

func (p *SecretStoreProvider) NewClient(ctx context.Context, store esv1.GenericStore, client client.Client, _ string) (esv1.SecretsClient, error) {
	converted, ok := store.(*tgtv1alpha1.VirtualMachine)
	if !ok {
		return nil, fmt.Errorf("target %q not found", store.GetObjectKind().GroupVersionKind().Kind)
	}
	uname, pass, err := getBasicAuth(ctx, client, converted.GetNamespace(), converted.Spec.Auth)
	if err != nil {
		return nil, err
	}
	cert, key, err := getCertAuth(ctx, client, converted.GetNamespace(), converted.Spec.Auth)
	if err != nil {
		return nil, err
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
		Namespace:         converted.GetNamespace(),
		KubeClient:        client,
	}, nil
}

func (s *ScanTarget) Lock() {
	mu.Lock()
}

func (s *ScanTarget) Unlock() {
	mu.Unlock()
}

func (s *ScanTarget) ScanForSecrets(ctx context.Context, regexes []string, threshold int) ([]scanv1alpha1.SecretInStoreRef, error) {
	u, err := url.Parse(s.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL %q: %w", s.URL, err)
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

func (s *ScanTarget) checkForJob(ctx context.Context, client *http.Client, jobID string) ([]scanv1alpha1.SecretInStoreRef, error) {
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

func (s *ScanTarget) runMatches(ctx context.Context, client *http.Client, jobID string) ([]scanv1alpha1.SecretInStoreRef, error) {
	matches, err := s.getJobMatches(ctx, client, jobID)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func (s *ScanTarget) getJobMatches(ctx context.Context, client *http.Client, jobID string) ([]scanv1alpha1.SecretInStoreRef, error) {
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
	secrets := []scanv1alpha1.SecretInStoreRef{}
	for _, match := range scanJobResponse.Match {
		secret := scanv1alpha1.SecretInStoreRef{
			APIVersion: tgtv1alpha1.SchemeGroupVersion.String(),
			Kind:       tgtv1alpha1.VirtualMachineKind,
			Name:       s.Name,
			RemoteRef: scanv1alpha1.RemoteRef{
				Key:      match.Key,
				Property: match.Property,
			},
		}
		secrets = append(secrets, secret)
	}
	return secrets, nil
}

func (s *ScanTarget) ScanForConsumers(ctx context.Context, location scanv1alpha1.SecretInStoreRef, hash string) ([]scanv1alpha1.ConsumerFinding, error) {
	u, err := url.Parse(s.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL %q: %w", s.URL, err)
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
				return nil, fmt.Errorf("loading client certificate: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
		client.Transport = &http.Transport{TLSClientConfig: tlsConfig}
	}

	reqBody := ConsumerRequest{
		Location: location,
		Paths:    s.Paths,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling consumer request: %w", err)
	}

	api := fmt.Sprintf("%s/api/v1/consumers?filePath=%s", s.URL, url.QueryEscape(location.RemoteRef.Key))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, api, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating consumer request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.AuthBasicUsername != nil && s.AuthBasicPassword != nil {
		req.SetBasicAuth(*s.AuthBasicUsername, *s.AuthBasicPassword)
	} else if s.AuthBearerToken != nil {
		req.Header.Set("Authorization", "Bearer "+*s.AuthBearerToken)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing consumer request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var scanResp []ConsumerScanJobResponse
	if err := json.NewDecoder(resp.Body).Decode(&scanResp); err != nil {
		return nil, fmt.Errorf("decoding consumer response: %w", err)
	}

	out := make([]scanv1alpha1.ConsumerFinding, 0, len(scanResp))
	for _, consumer := range scanResp {
		display := consumer.Comm
		if display == "" {
			display = consumer.Exe
		}

		var observedIndexTimestamp metav1.Time
		if !consumer.UpdatedAt.IsZero() {
			observedIndexTimestamp = metav1.NewTime(consumer.UpdatedAt.UTC())
		} else {
			observedIndexTimestamp = metav1.NewTime(consumer.CreatedAt.UTC())
		}

		out = append(out, scanv1alpha1.ConsumerFinding{
			ObservedIndex: scanv1alpha1.SecretUpdateRecord{
				Timestamp:  observedIndexTimestamp,
				SecretHash: hash,
			},
			Location:    location,
			Type:        tgtv1alpha1.VirtualMachineKind,
			ID:          stableConsumerID(consumer),
			DisplayName: display,
			Attributes: scanv1alpha1.ConsumerAttrs{
				VMProcess: &scanv1alpha1.VMProcessSpec{
					RUID:       int64(consumer.RUID),
					EUID:       int64(consumer.EUID),
					Executable: consumer.Exe,
					Cmdline:    consumer.Comm,
				},
			},
		})
	}
	return out, nil
}

type JobNotReadyErr struct{}

func (e JobNotReadyErr) Error() string {
	return "job not ready"
}

func init() {
	tgtv1alpha1.Register(tgtv1alpha1.VirtualMachineKind, &Provider{})
	esv1.RegisterByKind(&SecretStoreProvider{}, tgtv1alpha1.VirtualMachineKind, esv1.MaintenanceStatusMaintained)
}

func getBasicAuth(ctx context.Context, client client.Client, namespace string, auth *tgtv1alpha1.Authentication) (string, string, error) {
	var uname, pass string
	var err error
	if auth != nil {
		if auth.Basic != nil {
			uname, err = resolvers.SecretKeyRef(ctx, client, "", namespace, auth.Basic.UsernameSecretRef)
			if err != nil {
				return "", "", err
			}
			pass, err = resolvers.SecretKeyRef(ctx, client, "", namespace, auth.Basic.PasswordSecretRef)
			if err != nil {
				return "", "", err
			}
		}
	}
	return uname, pass, nil
}

func getCertAuth(ctx context.Context, client client.Client, namespace string, auth *tgtv1alpha1.Authentication) (string, string, error) {
	var cert, key string
	var err error
	if auth != nil {
		if auth.Certificate != nil {
			cert, err = resolvers.SecretKeyRef(ctx, client, "", namespace, auth.Certificate.ClientCertificateSecretRef)
			if err != nil {
				return "", "", err
			}
			key, err = resolvers.SecretKeyRef(ctx, client, "", namespace, auth.Certificate.ClientKeySecretRef)
			if err != nil {
				return "", "", err
			}
		}
	}
	return cert, key, nil
}

// stableConsumerID returns a stable external ID based on attributes that
// do not change across restarts. For VMs we avoid PID on purpose.
func stableConsumerID(consumer ConsumerScanJobResponse) string {
	base := consumer.Comm + "|" + consumer.Exe + "|" + fmt.Sprintf("%d|%d", consumer.RUID, consumer.EUID)
	sum := sha512.Sum512([]byte(base))
	return hex.EncodeToString(sum[:])
}
