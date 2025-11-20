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

package webhook

import (
	"context"
	"errors"
	"fmt"

	v1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider implements the webhook listener provider.
type Provider struct {
}

// CreateListener creates a new Listener that listens for webhook notifications based on the provided configuration and event channel.
func (p *Provider) CreateListener(ctx context.Context, config *v1alpha1.NotificationSource, client client.Client, eventChan chan events.SecretRotationEvent, logger logr.Logger) (schema.Listener, error) {
	if config == nil || config.Webhook == nil {
		return nil, errors.New("webhook config is nil")
	}
	server, err := createServer(config.Webhook)
	if err != nil {
		logger.Error(err, "failed to create webhook server")
		return nil, fmt.Errorf("failed to create webhook server: %w", err)
	}

	childCtx, cancel := context.WithCancel(ctx)

	listener := &Listener{
		config:     config.Webhook,
		eventChan:  eventChan,
		ctx:        childCtx,
		cancel:     cancel,
		logger:     logger,
		server:     server,
		client:     client,
		retryQueue: make(chan *RetryMessage),
	}

	listener.createHandler()

	return listener, nil
}

func init() {
	schema.RegisterProvider(schema.Webhook, &Provider{})
}
