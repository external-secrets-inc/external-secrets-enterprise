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

// Package gcp implements GCP authentication.
package gcp

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/util/resolvers"
)

const (
	// CloudPlatformRole is the GCP cloud platform role scope.
	CloudPlatformRole = "https://www.googleapis.com/auth/cloud-platform"
)

// NewTokenSource creates a new OAuth2 token source for GCP.
func NewTokenSource(ctx context.Context, auth *v1alpha1.GooglePubSubAuth, projectID string, kube kclient.Client) (oauth2.TokenSource, error) {
	if auth == nil {
		return google.DefaultTokenSource(ctx, CloudPlatformRole)
	}
	ts, err := tokenSourceFromSecret(ctx, auth, kube)
	if ts != nil || err != nil {
		return ts, err
	}
	if auth.WorkloadIdentity != nil && auth.WorkloadIdentity.ClusterProjectID != "" {
		projectID = auth.WorkloadIdentity.ClusterProjectID
	}
	wi, err := newWorkloadIdentity(ctx, projectID)
	if err != nil {
		return nil, errors.New("unable to initialize workload identity")
	}
	defer wi.Close() //nolint
	ts, err = wi.TokenSource(ctx, auth, kube)
	if ts != nil || err != nil {
		return ts, err
	}
	return google.DefaultTokenSource(ctx, CloudPlatformRole)
}

func tokenSourceFromSecret(ctx context.Context, auth *v1alpha1.GooglePubSubAuth, kube kclient.Client) (oauth2.TokenSource, error) {
	sr := auth.SecretRef
	if sr == nil {
		return nil, nil
	}
	credentials, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		&auth.SecretRef.SecretAccessKey)
	if err != nil {
		return nil, err
	}
	config, err := google.JWTConfigFromJSON([]byte(credentials), CloudPlatformRole)
	if err != nil {
		return nil, fmt.Errorf("unable to process jwt credentials:%w", err)
	}
	return config.TokenSource(ctx), nil
}
