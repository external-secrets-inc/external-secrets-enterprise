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

// Package listener manages event listeners for secret rotation.
package listener

import (
	"context"
	"crypto/sha3"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	esov1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Manager manages event listeners for secret rotation events. It coordinates the creation, starting, and stopping of listeners.
type Manager struct {
	context   context.Context
	client    client.Client
	eventChan chan events.SecretRotationEvent
	listeners map[types.NamespacedName]map[string]schema.Listener
	mu        sync.Mutex
	logger    logr.Logger
}

// NewListenerManager creates a new listener manager.
func NewListenerManager(ctx context.Context, eventChan chan events.SecretRotationEvent, client client.Client, logger logr.Logger) *Manager {
	return &Manager{
		context:   ctx,
		eventChan: eventChan,
		client:    client,
		listeners: make(map[types.NamespacedName]map[string]schema.Listener),
		logger:    logger,
	}
}

// ManageListeners manages the active listeners based on the provided notification sources. It starts new listeners and stops unwanted ones.
func (lm *Manager) ManageListeners(manifestName types.NamespacedName, sources []esov1alpha1.NotificationSource) error {
	lm.mu.Lock()
	// Register listener for that manifest if we haven't
	if _, ok := lm.listeners[manifestName]; !ok {
		lm.listeners[manifestName] = make(map[string]schema.Listener)
	}
	// Clean up desired listeners for manifest
	desiredListeners := map[string]esov1alpha1.NotificationSource{}
	defer lm.mu.Unlock()
	for _, source := range sources {
		key, err := generateListenerKey(source)
		if err != nil {
			lm.logger.Error(err, "failed to generate listener key", "source", source)
			continue
		}
		desiredListeners[key] = source
	}

	// Remove unwanted listeners
	for key, l := range lm.listeners[manifestName] {
		if _, exists := desiredListeners[key]; !exists {
			lm.logger.Info("Stopping listener", "key", key)
			if err := l.Stop(); err != nil {
				lm.logger.Error(err, "failed to stop listener", "key", key)
			}
			delete(lm.listeners[manifestName], key)
			lm.logger.V(1).Info("removing listener entry", "manifest", manifestName, "key", key)
		}
	}

	// Add new listeners
	for key, source := range desiredListeners {
		if _, exists := lm.listeners[manifestName][key]; !exists {
			lm.logger.Info("Creating new eventListener", "key", key, "type", source.Type)
			prov := schema.GetProvider(source.Type)
			if prov == nil {
				lm.logger.Error(nil, "failed to get provider", "type", source.Type)
				continue
			}
			eventListener, err := prov.CreateListener(lm.context, &source, lm.client, lm.eventChan, lm.logger)
			if err != nil {
				lm.logger.Error(err, "failed to create listener", "key", key)
				continue
			}
			if err := eventListener.Start(); err != nil {
				lm.logger.Error(err, "failed to start listener", "key", key)
				continue
			}
			lm.listeners[manifestName][key] = eventListener
		} else {
			lm.logger.V(1).Info("listener already exists", "key", key)
		}
	}
	// cleanup if empty
	if len(lm.listeners[manifestName]) == 0 {
		lm.logger.V(1).Info("removing listener map for manifest", "manifest", manifestName)
		delete(lm.listeners, manifestName)
	}
	return nil
}

// StopAll stops all active listeners managed by the Manager and removes them from the listeners map.
func (lm *Manager) StopAll() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	var errs []error
	for mk, mv := range lm.listeners {
		for key, l := range mv {
			lm.logger.Info("Stopping listener", "key", key)
			if err := l.Stop(); err != nil {
				lm.logger.Error(err, "failed to stop listener", "key", key)
				errs = append(errs, err)
			}
			delete(lm.listeners[mk], key)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// generateListenerKey creates a unique key for a NotificationSource based on its Type and configuration.
func generateListenerKey(source esov1alpha1.NotificationSource) (string, error) {
	// Marshal the specific configuration based on the Type
	var config any
	switch source.Type {
	case schema.AWSSQS:
		config = source.AwsSqs
	case schema.AzureEventGrid:
		config = source.AzureEventGrid
	case schema.GooglePubSub:
		config = source.GooglePubSub
	case schema.Webhook:
		config = source.Webhook
	case schema.HashicorpVault:
		config = source.HashicorpVault
	case schema.TCPSocket:
		config = source.TCPSocket
	case schema.Mock:
		config = source.Mock
	case schema.KubernetesSecret:
		config = source.KubernetesSecret
	default:
		return "", fmt.Errorf("unsupported notification source type: %s", source.Type)
	}

	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	// Compute SHA1 hash of the configuration for uniqueness
	hash := sha3.Sum224(data)

	// Combine Type and hash to form the key
	key := fmt.Sprintf("%s-%x", source.Type, hash)

	return key, nil
}
