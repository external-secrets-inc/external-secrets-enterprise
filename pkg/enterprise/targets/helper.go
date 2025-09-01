// Copyright External Secrets Inc. 2025
// All rights reserved

package targets

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const maxHistoryPerLocation = 20

func UpdateTargetPushIndex(
	ctx context.Context,
	kubeClient client.Client,
	name string,
	namespace string,
	key string,
	property string,
	hash string,
) error {
	if kubeClient == nil {
		return errors.New("kube client is not configured on ScanTarget")
	}

	locationKey := key
	if strings.TrimSpace(property) != "" {
		locationKey = fmt.Sprintf("%s.%s", key, property)
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var obj tgtv1alpha1.GithubRepository
		if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &obj); err != nil {
			return err
		}

		if obj.Status.PushIndex == nil {
			obj.Status.PushIndex = make(map[string][]tgtv1alpha1.SecretUpdateRecord, 1)
		}

		hist := obj.Status.PushIndex[locationKey]

		hist = append(hist, tgtv1alpha1.SecretUpdateRecord{
			Timestamp:  metav1.NewTime(metav1.Now().UTC()),
			SecretHash: hash,
		})

		if len(hist) > maxHistoryPerLocation {
			hist = hist[len(hist)-maxHistoryPerLocation:]
		}
		obj.Status.PushIndex[locationKey] = hist

		return kubeClient.Status().Update(ctx, &obj)
	})
}

func Hash(value []byte) string {
	hash := sha512.Sum512(value)
	return hex.EncodeToString(hash[:])
}
