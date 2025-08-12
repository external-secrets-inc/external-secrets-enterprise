package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/google/go-github/v74/github"
	corev1 "k8s.io/api/core/v1"
)

// PushSecret creates a pull request that replaces the exact old value (Property) in the file (Key).
func (s *ScanTarget) PushSecret(ctx context.Context, secret *corev1.Secret, remoteRef esv1.PushSecretData) error {
	if remoteRef.GetProperty() == "" || remoteRef.GetRemoteKey() == "" {
		return errors.New("remoteRef.Property and remoteRef.Key are mandatory")
	}
	newVal, ok := secret.Data[remoteRef.GetSecretKey()]
	if !ok {
		return fmt.Errorf("secret key %q not found in secret data", remoteRef.GetSecretKey())
	}
	indexes := remoteRef.GetProperty()
	filename := remoteRef.GetRemoteKey()

	gh, err := newGitHubClient(ctx, s.AuthToken, s.EnterpriseURL, s.UploadURL)
	if err != nil {
		return fmt.Errorf("error creating new GitHub client: %w", err)
	}
	owner, repo, baseBranch := s.Owner, s.Repo, s.Branch

	ref, _, err := gh.Git.GetRef(ctx, owner, repo, "refs/heads/"+baseBranch)
	if err != nil {
		return fmt.Errorf("error getting repository ref: %w", err)
	}
	newBranch := fmt.Sprintf("external-secrets-update-%d", time.Now().Unix())
	_, _, err = gh.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref: github.Ptr("refs/heads/" + newBranch),
		Object: &github.GitObject{
			SHA: ref.Object.SHA,
		},
	})
	if err != nil {
		return fmt.Errorf("error creating new branch: %w", err)
	}

	rc, _, _, err := gh.Repositories.GetContents(ctx, owner, repo, filename, &github.RepositoryContentGetOptions{Ref: baseBranch})
	if err != nil {
		return fmt.Errorf("error getting file contents: %w", err)
	}
	if rc == nil || rc.GetType() != "file" {
		return fmt.Errorf("path %q is not a file", filename)
	}
	content, err := rc.GetContent()
	if err != nil {
		if rc.Content != nil {
			if b, decErr := base64.StdEncoding.DecodeString(*rc.Content); decErr == nil {
				content = string(b)
			} else {
				return fmt.Errorf("error decoding file content: %w", decErr)
			}
		} else {
			return fmt.Errorf("empty file content")
		}
	}
	fileSHA := rc.GetSHA()

	var start, end int
	if _, err := fmt.Sscanf(indexes, "%d:%d", &start, &end); err != nil {
		return fmt.Errorf("invalid property format %q (expected \"start:end\"): %w", indexes, err)
	}
	if start < 0 || end < 0 || start >= end {
		return fmt.Errorf("invalid index range: %d:%d", start, end)
	}
	if end > len(content) {
		return fmt.Errorf("end index %d out of bounds (file length %d)", end, len(content))
	}

	var buf bytes.Buffer
	buf.WriteString(content[:start])
	buf.Write(newVal)
	buf.WriteString(content[end:])
	newContent := buf.Bytes()

	commitMsg := fmt.Sprintf("chore: update secret in %s", filename)
	_, _, err = gh.Repositories.UpdateFile(ctx, owner, repo, filename, &github.RepositoryContentFileOptions{
		Message: github.Ptr(commitMsg),
		Content: newContent,
		SHA:     github.Ptr(fileSHA),
		Branch:  github.Ptr(newBranch),
	})
	if err != nil {
		return fmt.Errorf("update file: %w", err)
	}

	title := fmt.Sprintf("Update secret in %s", filename)
	pr, _, err := gh.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
		Title: github.Ptr(title),
		Head:  github.Ptr(newBranch),
		Base:  github.Ptr(baseBranch),
		Body:  github.Ptr("This PR was created automatically by External Secrets to update a hardcoded secret."),
	})
	if err != nil {
		return fmt.Errorf("error creating PR: %w", err)
	}

	log.Printf("pull request created by push secret: %d", *pr.Number)
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
