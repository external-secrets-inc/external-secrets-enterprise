// Copyright External Secrets Inc. 2025
// All Rights Reserved

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

	"github.com/golang-jwt/jwt/v5"
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
	GitHubClient  *github.Client
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

	githubClient, err := newGitHubClient(ctx, token, converted.Spec.EnterpriseURL, converted.Spec.UploadURL, converted.Spec.CABundle)
	if err != nil {
		return nil, fmt.Errorf("error creating new GitHub client: %w", err)
	}

	branch, err := resolveBranch(ctx, githubClient, converted.Spec.Owner, converted.Spec.Repository, converted.Spec.Branch)
	if err != nil {
		return nil, fmt.Errorf("error setting repo branch: %w", err)
	}

	return &ScanTarget{
		Name:          converted.GetName(),
		Owner:         converted.Spec.Owner,
		Repo:          converted.Spec.Repository,
		Branch:        branch,
		Paths:         converted.Spec.Paths,
		EnterpriseURL: converted.Spec.EnterpriseURL,
		UploadURL:     converted.Spec.UploadURL,
		CABundle:      converted.Spec.CABundle,
		AuthToken:     token,
		GitHubClient:  githubClient,
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
		return nil, fmt.Errorf("error resolving github token: %w", err)
	}

	githubClient, err := newGitHubClient(ctx, token, converted.Spec.EnterpriseURL, converted.Spec.UploadURL, converted.Spec.CABundle)
	if err != nil {
		return nil, fmt.Errorf("error creating new GitHub client: %w", err)
	}

	branch, err := resolveBranch(ctx, githubClient, converted.Spec.Owner, converted.Spec.Repository, converted.Spec.Branch)
	if err != nil {
		return nil, fmt.Errorf("error setting repo branch: %w", err)
	}

	return &ScanTarget{
		Name:          converted.GetName(),
		Owner:         converted.Spec.Owner,
		Repo:          converted.Spec.Repository,
		Branch:        branch,
		Paths:         converted.Spec.Paths,
		EnterpriseURL: converted.Spec.EnterpriseURL,
		UploadURL:     converted.Spec.UploadURL,
		CABundle:      converted.Spec.CABundle,
		AuthToken:     token,
		GitHubClient:  githubClient,
	}, nil
}

func (s *ScanTarget) Scan(ctx context.Context, secrets []string, _ int) ([]tgtv1alpha1.SecretInStoreRef, error) {
	owner, repo, baseBranch := s.Owner, s.Repo, s.Branch

	ref, _, err := s.GitHubClient.Git.GetRef(ctx, owner, repo, "refs/heads/"+baseBranch)
	if err != nil {
		return nil, fmt.Errorf("error getting ref: %w", err)
	}
	commit, _, err := s.GitHubClient.Git.GetCommit(ctx, owner, repo, ref.GetObject().GetSHA())
	if err != nil {
		return nil, fmt.Errorf("error getting base commit: %w", err)
	}

	tree, _, err := s.GitHubClient.Git.GetTree(ctx, owner, repo, commit.GetTree().GetSHA(), true)
	if err != nil {
		return nil, fmt.Errorf("error getting tree: %w", err)
	}

	var results []tgtv1alpha1.SecretInStoreRef

	pathFilters := newPathFilter(s.Paths)

	for _, treeEntry := range tree.Entries {
		if treeEntry.GetType() != "blob" {
			continue
		}
		path := treeEntry.GetPath()
		if !pathFilters.allow(path) {
			continue
		}

		repositoryContent, _, _, err := s.GitHubClient.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{Ref: baseBranch})
		if err != nil || repositoryContent == nil || repositoryContent.GetType() != "file" {
			continue
		}

		var content string
		if repositoryContent.Content != nil {
			content, err = repositoryContent.GetContent()
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
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(ctx, tokenSource)

	if strings.TrimSpace(caBundle) != "" {
		client, err := httpClientWithCABundle(httpClient, caBundle)
		if err != nil {
			return nil, err
		}
		httpClient = client
	}

	apiBase := strings.TrimSpace(enterpriseURL)
	uploadBase := strings.TrimSpace(uploadURL)

	if apiBase == "" && uploadBase == "" {
		return github.NewClient(httpClient), nil
	}

	// Ensure trailing slashes per go-github expectations
	if apiBase != "" && !strings.HasSuffix(apiBase, "/") {
		apiBase += "/"
	}
	if uploadBase != "" && !strings.HasSuffix(uploadBase, "/") {
		uploadBase += "/"
	}

	return github.NewClient(httpClient).WithEnterpriseURLs(apiBase, uploadBase)
}

func resolveGithubToken(ctx context.Context, kube client.Client, githubRepository *tgtv1alpha1.GithubRepository) (string, error) {
	if githubRepository.Spec.Auth == nil {
		return "", fmt.Errorf("spec.auth is required")
	}

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
		return jwtToken, nil
	}

	return "", fmt.Errorf("spec.auth must define either token or appAuth")
}

func resolveBranch(ctx context.Context, githubClient *github.Client, owner, repo, baseBranch string) (string, error) {
	branch := strings.TrimSpace(baseBranch)
	if branch == "" {
		r, _, err := githubClient.Repositories.Get(ctx, owner, repo)
		if err != nil {
			return "", fmt.Errorf("get repository %s/%s: %w", owner, repo, err)
		}
		branch = r.GetDefaultBranch()
		if branch == "" {
			return "", fmt.Errorf("repository %s/%s has no default branch", owner, repo)
		}
	}
	return branch, nil
}

func readSecretKey(ctx context.Context, kube client.Client, namespace string, selector esmeta.SecretKeySelector) ([]byte, error) {
	// reuse resolver for consistency with project code style
	value, err := resolvers.SecretKeyRef(ctx, kube, resolvers.EmptyStoreKind, namespace, &selector)
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
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
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
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

	client := *base
	client.Transport = transport
	return &client, nil
}

func cloneTransport(roundTripper http.RoundTripper) *http.Transport {
	if roundTripper == nil {
		return &http.Transport{}
	}
	if transport, ok := roundTripper.(*http.Transport); ok {
		cp := transport.Clone()
		return cp
	}
	return &http.Transport{}
}

func cloneTLSConfig(config *tls.Config) *tls.Config {
	if config == nil {
		return &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	cp := config.Clone()
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
