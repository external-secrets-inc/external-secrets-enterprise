// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

/*
copyright External Secrets Inc. All Rights Reserved.
*/

package resolvers

import (
	"context"
	"fmt"

	v1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// EmptyStoreKind is used to determine if a store is cluster-scoped or not.
	// The EmptyStoreKind is not cluster-scoped, hence resources
	// cannot be resolved across namespaces.
	// TODO: when we implement cluster-scoped generators
	// we can remove this and replace it with a interface.
	EmptyStoreKind = "EmptyStoreKind"

	// ErrGetKubeSecret is the error message when a Kubernetes secret cannot be retrieved.
	errGetKubeSecret = "cannot get Kubernetes secret %q: %w"
	// ErrSecretKeyFmt is the error message when a secret key cannot be found.
	errSecretKeyFmt = "cannot find secret data for key: %q"
)

// SecretKeyRef resolves a metav1.SecretKeySelector and returns the value of the secret it points to.
// A user must pass the namespace of the originating ExternalSecret, as this may differ
// from the namespace defined in the SecretKeySelector.
// This func ensures that only a ClusterSecretStore is able to request secrets across namespaces.
func SecretKeyRef(
	ctx context.Context,
	c client.Client,
	ref *v1alpha1.SecretKeySelector) (string, error) {
	key := types.NamespacedName{
		Namespace: ref.Namespace,
		Name:      ref.Name,
	}
	secret := &corev1.Secret{}
	err := c.Get(ctx, key, secret)
	if err != nil {
		return "", fmt.Errorf(errGetKubeSecret, ref.Name, err)
	}
	val, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf(errSecretKeyFmt, ref.Key)
	}
	return string(val), nil
}
