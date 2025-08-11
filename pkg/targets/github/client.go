package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	corev1 "k8s.io/api/core/v1"
)

// PushSecret creates a pull request with the secret contents update committed.
func (s *ScanTarget) PushSecret(ctx context.Context, secret *corev1.Secret, remoteRef esv1.PushSecretData) error {
	if remoteRef.GetProperty() == "" || remoteRef.GetRemoteKey() == "" {
		return errors.New("remoteRef.Property and remoteRef.Key are mandatory")
	}
	if remoteRef.GetStartIndex() == nil || remoteRef.GetEndIndex() == nil {
		return errors.New("remoteRef must include StartIndex and EndIndex")
	}

	newValue, ok := secret.Data[remoteRef.GetSecretKey()]
	if !ok {
		return fmt.Errorf("secret key %q not found in secret data", remoteRef.GetSecretKey())
	}

	client := &http.Client{Timeout: 10 * time.Second}
	apiBase := fmt.Sprintf("%s/repos/%s/%s", strings.TrimSuffix(s.URL, "/"), s.Owner, s.Repo)

	// Step 1: Get latest file content from default branch
	contentsURL := fmt.Sprintf("%s/contents/%s?ref=%s", apiBase, remoteRef.GetRemoteKey(), s.Branch)
	fileMeta, err := getGitHubFile(ctx, client, contentsURL, s.AuthToken)
	if err != nil {
		return fmt.Errorf("getting file: %w", err)
	}
	rawContent, err := base64.StdEncoding.DecodeString(fileMeta.Content)
	if err != nil {
		return fmt.Errorf("base64 decode file content: %w", err)
	}

	start := *remoteRef.GetStartIndex()
	end := *remoteRef.GetEndIndex()
	if start < 0 || end > len(rawContent) || start >= end {
		return fmt.Errorf("invalid match index range: %d-%d", start, end)
	}

	// Step 2: Patch the content
	var updated bytes.Buffer
	updated.Write(rawContent[:start])
	updated.Write(newValue)
	updated.Write(rawContent[end:])

	// Step 3: Open a pull request with this updated content
	filename := remoteRef.GetRemoteKey()
	if err := s.createPullRequest(ctx, filename, updated.Bytes()); err != nil {
		return fmt.Errorf("creating pull request: %w", err)
	}

	return nil
}

// DeleteSecret is not implemented for GitHub target.
func (s *ScanTarget) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

// SecretExists is not implemented for GitHub target.
func (s *ScanTarget) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// GetSecret is not implemented for GitHub target.
func (s *ScanTarget) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, fmt.Errorf("not implemented - write-only target")
}

func (s *ScanTarget) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented - write-only target")
}

func (s *ScanTarget) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented - write-only target")
}

func (s *ScanTarget) Close(ctx context.Context) error {
	return nil
}

func (s *ScanTarget) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultUnknown, nil
}

func (s *ScanTarget) createPullRequest(ctx context.Context, filename string, patchedContent []byte) error {
	client := &http.Client{Timeout: 10 * time.Second}
	apiBase := fmt.Sprintf("%s/repos/%s/%s", strings.TrimSuffix(s.URL, "/"), s.Owner, s.Repo)

	// 1. Get the base branch SHA
	baseRefURL := fmt.Sprintf("%s/git/ref/heads/%s", apiBase, s.Branch)
	baseSHA, err := getBranchSHA(ctx, client, baseRefURL, s.AuthToken)
	if err != nil {
		return fmt.Errorf("getting base branch sha: %w", err)
	}

	// 2. Create a new branch
	newBranch := fmt.Sprintf("external-secrets-update-%d", time.Now().Unix())
	refURL := fmt.Sprintf("%s/git/refs", apiBase)
	newRef := map[string]string{
		"ref": fmt.Sprintf("refs/heads/%s", newBranch),
		"sha": baseSHA,
	}
	if err := postJSON(ctx, client, refURL, s.AuthToken, newRef, nil); err != nil {
		return fmt.Errorf("creating new branch: %w", err)
	}

	// 3. Create a blob for the patched content
	blobURL := fmt.Sprintf("%s/git/blobs", apiBase)
	var blobResp struct {
		SHA string `json:"sha"`
	}
	blobReq := map[string]string{
		"content":  string(patchedContent),
		"encoding": "utf-8",
	}
	if err := postJSON(ctx, client, blobURL, s.AuthToken, blobReq, &blobResp); err != nil {
		return fmt.Errorf("creating blob: %w", err)
	}

	// 4. Get the base tree from the latest commit
	commitURL := fmt.Sprintf("%s/git/commits/%s", apiBase, baseSHA)
	var commitResp struct {
		Tree struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}
	if err := getJSON(ctx, client, commitURL, s.AuthToken, &commitResp); err != nil {
		return fmt.Errorf("getting base commit tree: %w", err)
	}

	// 5. Create a tree with the updated file blob
	treeURL := fmt.Sprintf("%s/git/trees", apiBase)
	treeReq := map[string]interface{}{
		"base_tree": commitResp.Tree.SHA,
		"tree": []map[string]string{
			{
				"path": filename,
				"mode": "100644",
				"type": "blob",
				"sha":  blobResp.SHA,
			},
		},
	}
	var treeResp struct {
		SHA string `json:"sha"`
	}
	if err := postJSON(ctx, client, treeURL, s.AuthToken, treeReq, &treeResp); err != nil {
		return fmt.Errorf("creating tree: %w", err)
	}

	// 6. Create a commit
	commitReq := map[string]interface{}{
		"message": "chore: update secret in-place",
		"tree":    treeResp.SHA,
		"parents": []string{baseSHA},
	}
	var newCommitResp struct {
		SHA string `json:"sha"`
	}
	if err := postJSON(ctx, client, fmt.Sprintf("%s/git/commits", apiBase), s.AuthToken, commitReq, &newCommitResp); err != nil {
		return fmt.Errorf("creating commit: %w", err)
	}

	// 7. Update the branch to point to the new commit
	updateRefURL := fmt.Sprintf("%s/git/refs/heads/%s", apiBase, newBranch)
	updateRefReq := map[string]string{
		"sha": newCommitResp.SHA,
	}
	if err := patchJSON(ctx, client, updateRefURL, s.AuthToken, updateRefReq, nil); err != nil {
		return fmt.Errorf("updating new branch to commit: %w", err)
	}

	// 8. Create the pull request
	prURL := fmt.Sprintf("%s/pulls", apiBase)
	prReq := map[string]string{
		"title": fmt.Sprintf("Update secret in file %s", filename),
		"head":  newBranch,
		"base":  s.Branch,
		"body":  "This PR was created automatically by External Secrets to update a hardcoded secret.",
	}
	if err := postJSON(ctx, client, prURL, s.AuthToken, prReq, nil); err != nil {
		return fmt.Errorf("creating pull request: %w", err)
	}

	return nil
}

func getGitHubFile(ctx context.Context, client *http.Client, url, token string) (*GithubContentFile, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var file GithubContentFile
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return nil, err
	}
	if file.Type != "file" {
		return nil, fmt.Errorf("path is not a file")
	}
	return &file, nil
}

func getBranchSHA(ctx context.Context, client *http.Client, url, token string) (string, error) {
	var result struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := getJSON(ctx, client, url, token, &result); err != nil {
		return "", err
	}
	return result.Object.SHA, nil
}

func getJSON(ctx context.Context, client *http.Client, url, token string, out interface{}) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}

func postJSON(ctx context.Context, client *http.Client, url, token string, body, out interface{}) error {
	return doJSON(ctx, client, http.MethodPost, url, token, body, out)
}

func patchJSON(ctx context.Context, client *http.Client, url, token string, body, out interface{}) error {
	return doJSON(ctx, client, http.MethodPatch, url, token, body, out)
}

func doJSON(ctx context.Context, client *http.Client, method, url, token string, body, out interface{}) error {
	bts, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bts))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		buf, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error (%d): %s", resp.StatusCode, string(buf))
	}

	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
