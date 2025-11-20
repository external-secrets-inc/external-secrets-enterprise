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

// Package hashivault implements Hashicorp Vault listener.
package hashivault

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	v1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/tcp"
	vault "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/pkg/models/vault"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HashicorpVault represents a TCP socket listener configured to parse hashicorp vault messages.
// It utilizes a stop channel to manage its lifecycle.
type HashicorpVault struct {
	config    *v1alpha1.HashicorpVaultConfig
	context   context.Context
	cancel    context.CancelFunc
	client    client.Client
	eventChan chan events.SecretRotationEvent
	logger    logr.Logger
	tcpSocket *tcp.Socket
}

func (h *HashicorpVault) processFn(message []byte) {
	msg := &vault.AuditLog{}
	err := json.Unmarshal(message, msg)
	if err != nil {
		h.logger.Error(err, "Failed to unmarshal message", "Message", message)
		return
	}
	if !vault.ValidMessage(msg) {
		h.logger.V(1).Info("Invalid message - ignoring")
		return
	}
	basePath := msg.AuthResponse.MountPoint
	// Removing "data" if any
	path := strings.TrimPrefix(strings.Split(msg.AuthRequest.Path, basePath)[1], "data/")
	switch msg.AuthRequest.Operation {
	case "create":
	case "update":
		h.logger.V(1).Info("Received Valid Message", "Message", msg)
		event := events.SecretRotationEvent{
			SecretIdentifier:  path,
			RotationTimestamp: time.Now().Format("2006-01-02-15-04-05.000"),
			TriggerSource:     schema.HashicorpVault,
		}
		h.eventChan <- event
		h.logger.V(1).Info("Published event to eventChan", "Event", event)
	default:
		h.logger.V(2).Info("Non-Applicable Operation", "Operation", msg.AuthRequest.Operation)
	}
}

// Start initiates the HashicorpVault service, making it ready to accept incoming connections.
func (h *HashicorpVault) Start() error {
	return h.tcpSocket.Start()
}

// Stop stops the HashicorpVault  by closing the stop channel.
func (h *HashicorpVault) Stop() error {
	h.cancel()
	return h.tcpSocket.Stop()
}
