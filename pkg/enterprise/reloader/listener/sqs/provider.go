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

package sqs

import (
	"context"
	"errors"
	"fmt"

	"github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	listenerAWS "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/pkg/listener/aws"
	modelAWS "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/pkg/models/aws"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/util/mapper"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider implements the AWS SQS listener provider.
type Provider struct{}

// CreateListener creates a new AWSSQSListener.
func (p *Provider) CreateListener(ctx context.Context, config *v1alpha1.NotificationSource, client client.Client, eventChan chan events.SecretRotationEvent, logger logr.Logger) (schema.Listener, error) {
	if config == nil || config.AwsSqs == nil {
		return nil, errors.New("aws sqs config is nil")
	}
	// Create authenticated SQS Listener
	parsedConfig, err := mapper.TransformConfig[modelAWS.SQSConfig](config.AwsSqs)
	if err != nil {
		logger.Error(err, "Failed to parse config")
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	listener, err := listenerAWS.NewAWSSQSListener(ctx, &parsedConfig, client, logger)
	if err != nil {
		logger.Error(err, "Failed to create SQS Listener")
		return nil, fmt.Errorf("failed to create SQS Listener: %w", err)
	}

	return &AWSSQSListener{
		context:   ctx,
		listener:  listener,
		eventChan: eventChan,
		logger:    logger,
	}, nil
}

func init() {
	schema.RegisterProvider(schema.AWSSQS, &Provider{})
}
