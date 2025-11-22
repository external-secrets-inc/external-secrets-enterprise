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

// Package k8sconfigmap implements Kubernetes ConfigMap listener.
package k8sconfigmap

import (
	"context"
	"errors"
	"sync"

	"github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/kubernetes"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider implements the Kubernetes ConfigMap listener provider.
type Provider struct{}

// CreateListener creates a Kubernetes ConfigMap Listener.
func (p *Provider) CreateListener(ctx context.Context, config *v1alpha1.NotificationSource, client client.Client, eventChan chan events.SecretRotationEvent, logger logr.Logger) (schema.Listener, error) {
	if config == nil || config.KubernetesConfigMap == nil {
		return nil, errors.New("KubernetesConfigMap config is nil")
	}
	ctx, cancel := context.WithCancel(ctx)
	h := &kubernetes.Handler[*corev1.ConfigMap]{
		Config: &v1alpha1.KubernetesObjectConfig{
			ServerURL:     config.KubernetesConfigMap.ServerURL,
			Auth:          config.KubernetesConfigMap.Auth,
			LabelSelector: config.KubernetesConfigMap.LabelSelector,
		},
		Ctx:        ctx,
		Cancel:     cancel,
		Client:     client,
		EventChan:  eventChan,
		Logger:     logger,
		VersionMap: sync.Map{},
		Obj:        &corev1.ConfigMap{},
		Name:       "configmap",
	}

	return h, nil
}

func init() {
	schema.RegisterProvider(schema.KubernetesConfigMap, &Provider{})
}
