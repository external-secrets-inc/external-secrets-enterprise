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

// Copyright External Secrets Inc. 2025
// All Rights Reserved

// Package util provides utility functions for secret and token retrieval.
package util //nolint:revive,nolintlint

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetSecret retrieves a Kubernetes Secret.
func GetSecret(ctx context.Context, k8sClient client.Client, name, namespace string, logger logr.Logger) (*corev1.Secret, error) {
	logger.Info("Retrieving Kubernetes Secret", "SecretName", name, "Namespace", namespace)
	secret := &corev1.Secret{}
	key := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	if err := k8sClient.Get(ctx, key, secret); err != nil {
		logger.Error(err, "Failed to get secret", "SecretName", name, "Namespace", namespace)
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}
	logger.Info("Successfully retrieved secret", "SecretName", name)
	return secret, nil
}
