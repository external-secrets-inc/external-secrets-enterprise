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

package tcp

import (
	"context"
	"errors"

	v1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider implements the TCP socket listener provider.
type Provider struct{}

// CreateListener creates a new TCP socket listener.
func (p *Provider) CreateListener(ctx context.Context, config *v1alpha1.NotificationSource, client client.Client, eventChan chan events.SecretRotationEvent, logger logr.Logger) (schema.Listener, error) {
	if config == nil || config.TCPSocket == nil {
		return nil, errors.New("tcp socket config is nil")
	}
	ctx, cancel := context.WithCancel(ctx)
	h := &Socket{
		config:    config.TCPSocket,
		context:   ctx,
		cancel:    cancel,
		client:    client,
		eventChan: eventChan,
		logger:    logger,
	}
	h.SetProcessFn(h.processFn)
	return h, nil
}

// NewTCPSocketListener initializes a new TCP socket in a way other components can consume.
func NewTCPSocketListener(ctx context.Context, config *v1alpha1.TCPSocketConfig, client client.Client, eventChan chan events.SecretRotationEvent, logger logr.Logger) (*Socket, error) {
	ctx, cancel := context.WithCancel(ctx)
	sock := &Socket{
		config:    config,
		context:   ctx,
		cancel:    cancel,
		client:    client,
		eventChan: eventChan,
		logger:    logger,
	}
	sock.SetProcessFn(sock.defaultProcess)
	return sock, nil
}

func init() {
	schema.RegisterProvider(schema.TCPSocket, &Provider{})
}
