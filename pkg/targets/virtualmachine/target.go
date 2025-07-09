package virtualmachine

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"

	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Provider struct{}

type ScanTarget struct {
	URL               string
	CABundle          []byte
	AuthBasicUsername *string
	AuthBasicPassword *string
	AuthBearerToken   *string
	AuthClientCert    []byte
	AuthClientKey     []byte
}

func (p *Provider) NewClient(ctx context.Context, client client.Client, target client.Object) (*ScanTarget, error) {
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
		CABundle:          converted.Spec.CABundle,
		AuthBasicUsername: &uname,
		AuthBasicPassword: &pass,
		AuthClientCert:    []byte(cert),
		AuthClientKey:     []byte(key),
	}, nil
}

func (s *ScanTarget) Scan(ctx context.Context, regexes []string) ([]tgtv1alpha1.SecretInStoreRef, error) {
	u, err := url.Parse(s.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing URL %q: %w", s.URL, err)
	}

	client := &http.Client{}

	if u.Scheme == "https" {
		// 1. Create TLS config if CA bundle is provided
		tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12} //nolint
		if len(s.CABundle) > 0 {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(s.CABundle)
			tlsConfig.RootCAs = caCertPool
		}

		// 2. Configure mTLS if client cert and key are provided
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

	// 4. Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// 5. Add authentication header
	if s.AuthBasicUsername != nil && s.AuthBasicPassword != nil {
		req.SetBasicAuth(*s.AuthBasicUsername, *s.AuthBasicPassword)
	} else if s.AuthBearerToken != nil {
		req.Header.Set("Authorization", "Bearer "+*s.AuthBearerToken)
	}

	// TODO: execute request and process response
	_ = client
	_ = req

	return nil, nil
}
