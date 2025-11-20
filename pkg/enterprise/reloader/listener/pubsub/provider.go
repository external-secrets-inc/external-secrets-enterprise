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

// Package pubsub implements Google Pub/Sub listener.
package pubsub

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/pubsub" //nolint:staticcheck
	v1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/pkg/auth/gcp"
	"github.com/go-logr/logr"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider implements the Google Pub/Sub listener provider.
type Provider struct{}

// CreateListener creates a new GooglePubSubListener.
func (p *Provider) CreateListener(ctx context.Context, config *v1alpha1.NotificationSource, client client.Client, eventChan chan events.SecretRotationEvent, logger logr.Logger) (schema.Listener, error) {
	if config == nil || config.GooglePubSub == nil {
		return nil, errors.New("GooglePubSub config is nil")
	}
	ctx, cancel := context.WithCancel(ctx)

	ts, err := gcp.NewTokenSource(ctx, config.GooglePubSub.Auth, config.GooglePubSub.ProjectID, client)
	if err != nil {
		defer cancel()
		return nil, fmt.Errorf("could not create token source: %w", err)
	}

	pubsubClient, err := pubsub.NewClient(ctx, config.GooglePubSub.ProjectID, option.WithTokenSource(ts))
	if err != nil {
		defer cancel()
		return nil, fmt.Errorf("could not create pubsub client: %w", err)
	}
	return &GooglePubSub{
		config:       config.GooglePubSub,
		context:      ctx,
		cancel:       cancel,
		client:       client,
		eventChan:    eventChan,
		logger:       logger,
		pubsubClient: pubsubClient,
	}, nil
}

func init() {
	schema.RegisterProvider(schema.GooglePubSub, &Provider{})
}
