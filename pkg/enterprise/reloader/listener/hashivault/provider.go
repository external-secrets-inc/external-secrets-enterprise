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

package hashivault

import (
	"context"
	"errors"

	v1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/tcp"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider implements the Hashicorp Vault listener provider.
type Provider struct{}

// CreateListener initializes a new TCP socket listener using the provided configuration and event channel.
func (p *Provider) CreateListener(ctx context.Context, config *v1alpha1.NotificationSource, client client.Client, eventChan chan events.SecretRotationEvent, logger logr.Logger) (schema.Listener, error) {
	if config == nil || config.HashicorpVault == nil {
		return nil, errors.New("HashicorpVault config is nil")
	}
	ctx, cancel := context.WithCancel(ctx)
	h := &HashicorpVault{
		config:    config.HashicorpVault,
		context:   ctx,
		cancel:    cancel,
		client:    client,
		eventChan: eventChan,
		logger:    logger,
	}

	sockConfig := &v1alpha1.TCPSocketConfig{
		Host: config.HashicorpVault.Host,
		Port: config.HashicorpVault.Port,
	}
	sock, err := tcp.NewTCPSocketListener(ctx, sockConfig, client, eventChan, logger)
	if err != nil {
		return nil, err
	}
	sock.SetProcessFn(h.processFn)
	h.tcpSocket = sock
	return h, nil
}

func init() {
	schema.RegisterProvider(schema.HashicorpVault, &Provider{})
}
