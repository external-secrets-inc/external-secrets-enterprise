package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/go-github/v74/github"
	"golang.org/x/oauth2"

	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
)

type ScanTarget struct {
	Name      string
	Owner     string
	Repo      string
	Branch    string // base branch to open the PR against
	Paths     []string
	URL       string // GitHub API URL (e.g. https://api.github.com)
	AuthToken string // GitHub token (App or PAT)
}

const (
	errNotImplemented    = "not implemented"
	errPropertyMandatory = "property is mandatory"
)

func (s *ScanTarget) Scan(ctx context.Context, secrets []string, _ int) ([]tgtv1alpha1.SecretInStoreRef, error) {
	// secrets == list of exact strings to search for
	gh := newGHClient(ctx, s.AuthToken)
	owner, repo, baseBranch := s.Owner, s.Repo, s.Branch

	// 1) Get base ref (commit SHA) and tree SHA
	ref, _, err := gh.Git.GetRef(ctx, owner, repo, "refs/heads/"+baseBranch)
	if err != nil {
		return nil, fmt.Errorf("get ref: %w", err)
	}
	commit, _, err := gh.Git.GetCommit(ctx, owner, repo, ref.GetObject().GetSHA())
	if err != nil {
		return nil, fmt.Errorf("get base commit: %w", err)
	}

	// 2) Get full tree recursively
	tree, _, err := gh.Git.GetTree(ctx, owner, repo, commit.GetTree().GetSHA(), true)
	if err != nil {
		return nil, fmt.Errorf("get tree: %w", err)
	}

	var results []tgtv1alpha1.SecretInStoreRef

PathLoop:
	for _, p := range s.Paths {
		prefix := strings.TrimPrefix(strings.TrimSpace(p), "/")
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}

		for _, te := range tree.Entries {
			if te.GetType() != "blob" {
				continue
			}
			path := te.GetPath()
			// filter by configured path (allow exact file or directory prefix)
			if prefix != "" && !(path == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(path, prefix)) {
				continue
			}

			// 3) Get file content (decoded)
			// Use Repos.GetContents to retrieve decoded content + file SHA if needed later
			rc, _, _, err := gh.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{Ref: baseBranch})
			if err != nil || rc == nil || rc.GetType() != "file" {
				continue
			}
			content, err := rc.GetContent()
			if err != nil {
				// fallback: if Content empty, rc.Content may be base64 encoded
				if rc.Content != nil {
					if b, decErr := base64.StdEncoding.DecodeString(*rc.Content); decErr == nil {
						content = string(b)
					} else {
						continue
					}
				} else {
					continue
				}
			}

			for _, secret := range secrets {
				if secret == "" {
					continue
				}
				idx := strings.Index(content, secret)
				if idx == -1 {
					continue
				}
				// Found — we return a SecretInStoreRef keyed by file path and property=the exact match
				results = append(results, tgtv1alpha1.SecretInStoreRef{
					APIVersion: tgtv1alpha1.SchemeGroupVersion.String(),
					Kind:       tgtv1alpha1.GithubTargetKind,
					Name:       s.Name,
					RemoteRef: tgtv1alpha1.RemoteRef{
						Key:      path,   // file path
						Property: secret, // exact old value; we'll use this in Push
					},
				})
				// if you want only 1 match per file per secret, continue outer loops
				continue PathLoop
			}
		}
	}

	return results, nil
}

func newGHClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	// For GH Enterprise, you can set BaseURL/UploadURL; otherwise default:
	client := github.NewClient(tc)
	// If s.URL points to GH Enterprise API, set it here:
	// client, _ = github.NewEnterpriseClient(apiBaseURL, uploadURL, tc)
	return client
}
