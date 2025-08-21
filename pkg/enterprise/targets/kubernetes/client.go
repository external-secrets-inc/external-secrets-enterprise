// Copyright External Secrets Inc. 2025
// All Rights Reserved
package kubernetes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *ScanTarget) PushSecret(ctx context.Context, secret *corev1.Secret, remoteRef esv1.PushSecretData) error {
	if secret == nil {
		return fmt.Errorf("secret is nil")
	}
	if remoteRef.GetRemoteKey() == "" || remoteRef.GetProperty() == "" {
		return fmt.Errorf("remoteRef.key and remoteRef.property are mandatory")
	}

	var newVal []byte
	var ok bool
	if remoteRef.GetSecretKey() == "" {
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

	namespace, name, err := parseNamespaceName(remoteRef.GetRemoteKey())
	if err != nil {
		return fmt.Errorf("invalid remote key %q: %w", remoteRef.GetRemoteKey(), err)
	}
	dataKey := strings.TrimSpace(remoteRef.GetProperty())

	var destination corev1.Secret
	err = s.KubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &destination)
	switch {
	case apierrors.IsNotFound(err):
		destination = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Data: map[string][]byte{dataKey: append([]byte(nil), newVal...)},
		}

		return s.KubeClient.Create(ctx, &destination)

	case err != nil:
		return err

	default:
		if destination.Data == nil {
			destination.Data = map[string][]byte{}
		}
		cur := destination.Data[dataKey]
		if bytes.Equal(cur, newVal) {
			return nil
		}
		destination.Data[dataKey] = append([]byte(nil), newVal...)

		return s.KubeClient.Update(ctx, &destination)
	}
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
	if s.KubeClient == nil {
		return esv1.ValidationResultError, fmt.Errorf("kube client is nil")
	}

	for _, p := range s.NamespaceInclude {
		if _, err := path.Match(p, "dummy"); err != nil {
			return esv1.ValidationResultError, fmt.Errorf("invalid include pattern %q: %w", p, err)
		}
	}
	for _, p := range s.NamespaceExclude {
		if _, err := path.Match(p, "dummy"); err != nil {
			return esv1.ValidationResultError, fmt.Errorf("invalid exclude pattern %q: %w", p, err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// List namespaces, then filter by include/exclude to verify scope isn’t empty
	var namespaceList corev1.NamespaceList
	if err := s.KubeClient.List(ctx, &namespaceList); err != nil {
		return esv1.ValidationResultError, fmt.Errorf("list namespaces: %w", err)
	}
	allowed := make([]string, 0, len(namespaceList.Items))
	for i := range namespaceList.Items {
		namespace := namespaceList.Items[i].Name
		if s.namespaceAllowed(namespace) {
			allowed = append(allowed, namespace)
		}
	}
	if len(s.NamespaceInclude) > 0 && len(allowed) == 0 {
		return esv1.ValidationResultError, fmt.Errorf("no namespaces matched include/exclude filters")
	}

	// Pick one namespace to probe permissions; prefer "default" if allowed.
	probeNamespace := pickProbeNamespace(allowed)
	if probeNamespace == "" {
		probeNamespace = "default"
	}

	var podList corev1.PodList
	if err := s.KubeClient.List(ctx, &podList, &crclient.ListOptions{
		Namespace:     probeNamespace,
		LabelSelector: s.SelectorOrEverything(),
	}); err != nil {
		return esv1.ValidationResultError, fmt.Errorf("list pods in %q: %w", probeNamespace, err)
	}

	var secList corev1.SecretList
	if err := s.KubeClient.List(ctx, &secList, &crclient.ListOptions{Namespace: probeNamespace}); err != nil {
		return esv1.ValidationResultError, fmt.Errorf("list secrets in %q: %w", probeNamespace, err)
	}

	var dummy corev1.Secret
	if err := s.KubeClient.Get(ctx, types.NamespacedName{Namespace: probeNamespace, Name: "ese-validate-nonexistent"}, &dummy); err != nil && apierrors.IsForbidden(err) {
		return esv1.ValidationResultError, fmt.Errorf("forbidden to get secrets in %q (need get/list for scan): %w", probeNamespace, err)
	}

	dryRunSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    probeNamespace,
			GenerateName: "ese-validate-",
			Labels:       map[string]string{"managed-by": "external-secrets-enterprise"},
		},
		Data: map[string][]byte{"_probe": []byte("ok")},
	}
	if err := s.KubeClient.Create(ctx, dryRunSecret, &crclient.CreateOptions{
		DryRun: []string{metav1.DryRunAll},
	}); err != nil {
		if apierrors.IsForbidden(err) {
			return esv1.ValidationResultError, fmt.Errorf("forbidden to create secrets in %q (need create permission for PushSecret): %w", probeNamespace, err)
		}
		return esv1.ValidationResultError, fmt.Errorf("dry-run create secret in %q failed: %w", probeNamespace, err)
	}

	return esv1.ValidationResultReady, nil
}

func pickProbeNamespace(allowed []string) string {
	if len(allowed) == 0 {
		return ""
	}
	for _, ns := range allowed {
		if ns == "default" {
			return ns
		}
	}
	return allowed[0]
}
