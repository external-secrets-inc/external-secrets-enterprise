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

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TokenRetriever retrieves service account tokens.
type TokenRetriever struct {
	k8sClient               client.Client
	serviceAccountName      string
	serviceAccountNamespace string
	audiences               []string
	logger                  logr.Logger
}

// NewTokenRetriever creates a new TokenRetriever.
func NewTokenRetriever(k8sClient client.Client, logger logr.Logger, serviceAccountName, serviceAccountNamespace string) *TokenRetriever {
	return &TokenRetriever{
		k8sClient:               k8sClient,
		serviceAccountName:      serviceAccountName,
		serviceAccountNamespace: serviceAccountNamespace,
		logger:                  logger,
		audiences:               []string{"sts.amazonaws.com"},
	}
}

// GetServiceAccountToken retrieves a service account token.
func (tr *TokenRetriever) GetServiceAccountToken() ([]byte, error) {
	tr.logger.Info("Attempting to retrieve service account token",
		"ServiceAccount", tr.serviceAccountName,
		"Namespace", tr.serviceAccountNamespace,
		"Audiences", tr.audiences,
	)

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tr.serviceAccountName,
			Namespace: tr.serviceAccountNamespace,
		},
	}

	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences: tr.audiences,
		},
	}

	err := tr.k8sClient.SubResource("token").Create(context.Background(), serviceAccount, tokenRequest)
	if err != nil {
		tr.logger.Error(err, "Error creating service account token",
			"ServiceAccount", tr.serviceAccountName,
			"Namespace", tr.serviceAccountNamespace,
		)
		return nil, fmt.Errorf("error creating service account token: %w", err)
	}

	tr.logger.Info("Successfully retrieved service account token",
		"ServiceAccount", tr.serviceAccountName,
		"Namespace", tr.serviceAccountNamespace,
	)

	return []byte(tokenRequest.Status.Token), nil
}

// GetIdentityToken retrieves an identity token.
func (tr *TokenRetriever) GetIdentityToken() ([]byte, error) {
	tr.logger.Info("Attempting to retrieve identity token")

	token, err := tr.GetServiceAccountToken()
	if err != nil {
		tr.logger.Error(err, "Error retrieving identity token")
		return nil, err
	}

	tr.logger.Info("Successfully retrieved identity token")

	return token, nil
}
