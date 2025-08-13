package github

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/go-github/v74/github"
	"github.com/labstack/gommon/log"
	"golang.org/x/oauth2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Provider struct{}
type ScanTarget struct {
	Name          string
	Owner         string
	Repo          string
	Branch        string // base branch to open the PR against
	Paths         []string
	EnterpriseURL string // GitHub API URL (e.g. http(s)://[hostname]/api/v3/)
	UploadURL     string // GitHub API Upload URL (e.g. http(s)://[hostname]/api/uploads/)
	CABundle      string // CA bundle for enterprise https
	AuthToken     string // GitHub token (App or PAT)
}

const (
	errNotImplemented    = "not implemented"
	errPropertyMandatory = "property is mandatory"
)

func (p *Provider) NewClient(ctx context.Context, client client.Client, target client.Object) (tgtv1alpha1.ScanTarget, error) {
	converted, ok := target.(*tgtv1alpha1.GithubRepository)
	if !ok {
		return nil, fmt.Errorf("target %q not found", target.GetObjectKind().GroupVersionKind().Kind)
	}

	// Resolve auth token: PAT or GitHub App Installation token
	token, err := resolveGithubToken(ctx, client, converted)
	if err != nil {
		return nil, fmt.Errorf("resolve github token: %w", err)
	}

	return &ScanTarget{
		Name:          converted.GetName(),
		Owner:         converted.Spec.Owner,
		Repo:          converted.Spec.Repository,
		Branch:        converted.Spec.Branch,
		Paths:         converted.Spec.Paths,
		EnterpriseURL: converted.Spec.EnterpriseURL,
		UploadURL:     converted.Spec.UploadURL,
		AuthToken:     token,
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
	converted, ok := store.(*tgtv1alpha1.GithubRepository)
	if !ok {
		return nil, fmt.Errorf("target %q not found", store.GetObjectKind().GroupVersionKind().Kind)
	}

	// Resolve auth token: PAT or GitHub App Installation token
	token, err := resolveGithubToken(ctx, client, converted)
	if err != nil {
		return nil, fmt.Errorf("resolve github token: %w", err)
	}

	return &ScanTarget{
		Name:          converted.GetName(),
		Owner:         converted.Spec.Owner,
		Repo:          converted.Spec.Repository,
		Branch:        converted.Spec.Branch,
		Paths:         converted.Spec.Paths,
		EnterpriseURL: converted.Spec.EnterpriseURL,
		UploadURL:     converted.Spec.UploadURL,
		AuthToken:     token,
	}, nil
}

func (s *ScanTarget) Scan(ctx context.Context, secrets []string, _ int) ([]tgtv1alpha1.SecretInStoreRef, error) {
	gh, err := newGitHubClient(ctx, s.AuthToken, s.EnterpriseURL, s.UploadURL, s.CABundle)
	if err != nil {
		return nil, fmt.Errorf("error creating new GitHub client: %w", err)
	}
	owner, repo, baseBranch := s.Owner, s.Repo, s.Branch

	ref, _, err := gh.Git.GetRef(ctx, owner, repo, "refs/heads/"+baseBranch)
	if err != nil {
		return nil, fmt.Errorf("get ref: %w", err)
	}
	commit, _, err := gh.Git.GetCommit(ctx, owner, repo, ref.GetObject().GetSHA())
	if err != nil {
		return nil, fmt.Errorf("get base commit: %w", err)
	}

	tree, _, err := gh.Git.GetTree(ctx, owner, repo, commit.GetTree().GetSHA(), true)
	if err != nil {
		return nil, fmt.Errorf("get tree: %w", err)
	}

	var results []tgtv1alpha1.SecretInStoreRef

	pathFilters := newPathFilter(s.Paths)

	for _, te := range tree.Entries {
		if te.GetType() != "blob" {
			continue
		}
		path := te.GetPath()
		// filter by configured path (allow exact file or directory prefix)
		if !pathFilters.allow(path) {
			continue
		}

		// 3) Get file content (decoded)
		// Use Repos.GetContents to retrieve decoded content + file SHA if needed later
		rc, _, _, err := gh.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{Ref: baseBranch})
		if err != nil || rc == nil || rc.GetType() != "file" {
			continue
		}

		var content string
		if rc.Content != nil {
			content, err = rc.GetContent()
			if err != nil {
				log.Errorf("error decoding github repository content from path %s: %w", path, err)
				continue
			}
		} else {
			log.Printf("file content is empty or not available directly (e.g., for directories).")
			continue
		}

		for _, secret := range secrets {
			if secret == "" {
				continue
			}
			idx := strings.Index(content, secret)
			if idx == -1 {
				continue
			}
			start := idx
			end := idx + len(secret)

			results = append(results, tgtv1alpha1.SecretInStoreRef{
				APIVersion: tgtv1alpha1.SchemeGroupVersion.String(),
				Kind:       tgtv1alpha1.GithubTargetKind,
				Name:       s.Name,
				RemoteRef: tgtv1alpha1.RemoteRef{
					Key:      path,                             // file path
					Property: fmt.Sprintf("%d:%d", start, end), // start:end format
				},
			})
		}
	}

	return results, nil
}

func newGitHubClient(ctx context.Context, token, enterpriseURL, uploadURL, caBundle string) (*github.Client, error) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)

	if strings.TrimSpace(caBundle) != "" {
		c, err := httpClientWithCABundle(tc, caBundle)
		if err != nil {
			return nil, err
		}
		tc = c
	}

	apiBase := strings.TrimSpace(enterpriseURL)
	uploadBase := strings.TrimSpace(uploadURL)

	if apiBase == "" && uploadBase == "" {
		return github.NewClient(tc), nil
	}

	// Ensure trailing slashes per go-github expectations
	if apiBase != "" && !strings.HasSuffix(apiBase, "/") {
		apiBase += "/"
	}
	if uploadBase != "" && !strings.HasSuffix(uploadBase, "/") {
		uploadBase += "/"
	}

	return github.NewClient(tc).WithEnterpriseURLs(apiBase, uploadBase)
}

func resolveGithubToken(ctx context.Context, kube client.Client, githubRepository *tgtv1alpha1.GithubRepository) (string, error) {
	if githubRepository.Spec.Auth == nil {
		return "", fmt.Errorf("spec.auth is required")
	}

	// 1) Personal Access Token
	if githubRepository.Spec.Auth.Token != nil {
		pat, err := resolvers.SecretKeyRef(ctx, kube, "", githubRepository.Namespace, &esmeta.SecretKeySelector{
			Namespace: &githubRepository.Namespace,
			Name:      githubRepository.Spec.Auth.Token.Name,
			Key:       githubRepository.Spec.Auth.Token.Key,
		})
		if err != nil {
			return "", fmt.Errorf("read PAT from secret: %w", err)
		}
		if pat == "" {
			return "", fmt.Errorf("empty PAT from secret")
		}
		return pat, nil
	}

	// 2) GitHub App: use private key to mint JWT, then exchange for an installation token
	if githubRepository.Spec.Auth.AppAuth != nil {
		pem, err := readSecretKey(ctx, kube, githubRepository.Namespace, esmeta.SecretKeySelector{
			Namespace: &githubRepository.Namespace,
			Name:      githubRepository.Spec.Auth.AppAuth.PrivateKey.Name,
			Key:       githubRepository.Spec.Auth.AppAuth.PrivateKey.Key,
		})
		if err != nil {
			return "", fmt.Errorf("read app private key: %w", err)
		}
		jwtToken, err := signAppJWT(pem, githubRepository.Spec.Auth.AppAuth.AppID)
		if err != nil {
			return "", fmt.Errorf("sign app jwt: %w", err)
		}
		instID := githubRepository.Spec.Auth.AppAuth.InstallID
		token, err := createInstallationToken(ctx, jwtToken, instID, githubRepository.Spec.EnterpriseURL, githubRepository.Spec.UploadURL, githubRepository.Spec.CABundle)
		if err != nil {
			return "", fmt.Errorf("create installation token: %w", err)
		}
		return token, nil
	}

	return "", fmt.Errorf("spec.auth must define either token or appAuth")
}

func readSecretKey(ctx context.Context, kube client.Client, ns string, sel esmeta.SecretKeySelector) ([]byte, error) {
	// reuse resolver for consistency with project code style
	val, err := resolvers.SecretKeyRef(ctx, kube, resolvers.EmptyStoreKind, ns, &sel)
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

// signAppJWT creates a short-lived JWT used for GitHub App authentication.
func signAppJWT(privateKeyPEM []byte, appID string) (string, error) {
	key, err := jwt.ParseRSAPrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("parse rsa key: %w", err)
	}
	claims := jwt.RegisteredClaims{
		Issuer:    appID,
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-10 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

// createInstallationToken exchanges the App JWT for an installation access token using go-github.
// If URL is a GHE API base, the client will target that host.
func createInstallationToken(ctx context.Context, appJWT, installID, enterpriseURL, uploadURL, caBundle string) (string, error) {
	ghClient, err := newGitHubClient(ctx, appJWT, enterpriseURL, uploadURL, caBundle)
	if err != nil {
		return "", fmt.Errorf("error creating new GitHub client: %w", err)
	}

	inst, err := parseInstallationID(installID)
	if err != nil {
		return "", err
	}
	token, _, err := ghClient.Apps.CreateInstallationToken(ctx, inst, &github.InstallationTokenOptions{})
	if err != nil {
		return "", fmt.Errorf("error creating apps installationToken: %w", err)
	}
	if token == nil || token.GetToken() == "" {
		return "", fmt.Errorf("empty installation token")
	}
	return token.GetToken(), nil
}

// Convert installation ID to int64 for the SDK
func parseInstallationID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid installID %q: %w", s, err)
	}
	return id, nil
}

func httpClientWithCABundle(base *http.Client, pemBundle string) (*http.Client, error) {
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	// Accept multiple concatenated PEM blocks
	ok := false
	rest := []byte(pemBundle)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			ok = pool.AppendCertsFromPEM(pem.EncodeToMemory(block))
		}
	}
	if !ok {
		// Try appending raw once if decode failed (single PEM)
		if !pool.AppendCertsFromPEM([]byte(pemBundle)) {
			return nil, fmt.Errorf("unable to append CA bundle")
		}
	}
	transport := cloneTransport(base.Transport)
	transport.TLSClientConfig = cloneTLSConfig(transport.TLSClientConfig)
	transport.TLSClientConfig.RootCAs = pool

	c := *base
	c.Transport = transport
	return &c, nil
}

func cloneTransport(rt http.RoundTripper) *http.Transport {
	if rt == nil {
		return &http.Transport{}
	}
	if t, ok := rt.(*http.Transport); ok {
		cp := t.Clone()
		return cp
	}
	// Wrap unknown round trippers
	return &http.Transport{}
}

func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	if cfg == nil {
		return &tls.Config{}
	}
	cp := cfg.Clone()
	return cp
}

type JobNotReadyErr struct{}

func (e JobNotReadyErr) Error() string {
	return "job not ready"
}

func init() {
	tgtv1alpha1.Register(tgtv1alpha1.GithubTargetKind, &Provider{})
	esv1.RegisterByKind(&SecretStoreProvider{}, tgtv1alpha1.GithubTargetKind, esv1.MaintenanceStatusMaintained)
}
