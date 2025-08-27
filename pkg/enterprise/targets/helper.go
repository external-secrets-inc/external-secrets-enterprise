package targets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateTargetPushIndex(
	ctx context.Context,
	kubeClient client.Client,
	name string,
	namespace string,
	key string,
	property string,
	meta map[string]any,
) error {
	if kubeClient == nil {
		return errors.New("kube client is not configured on ScanTarget")
	}

	metaRaw, _ := json.Marshal(meta)

	locationKey := key
	if strings.TrimSpace(property) != "" {
		locationKey = fmt.Sprintf("%s.%s", key, property)
	}

	// Patch status with conflict-retry
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var obj tgtv1alpha1.GithubRepository
		if err := kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &obj); err != nil {
			return err
		}
		if obj.Status.PushIndex == nil {
			obj.Status.PushIndex = make(map[string]tgtv1alpha1.SecretUpdateRecord, 1)
		}
		obj.Status.PushIndex[locationKey] = tgtv1alpha1.SecretUpdateRecord{
			Timestamp: metav1.Now(),
			Metadata:  apiextensionsv1.JSON{Raw: metaRaw},
		}
		return kubeClient.Status().Update(ctx, &obj)
	})
}
