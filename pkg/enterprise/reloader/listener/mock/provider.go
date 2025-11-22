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

package mock

import (
	"context"
	"errors"
	"time"

	v1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/schema"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Provider struct{}

// CreateListener creates a mock listener for simulated secret rotation events.
func (p *Provider) CreateListener(ctx context.Context, source *v1alpha1.NotificationSource, client client.Client, eventChan chan events.SecretRotationEvent, logger logr.Logger) (schema.Listener, error) {
	if source == nil || source.Mock == nil {
		return nil, errors.New("mock listener requires a valid mock configuration")
	}
	mockEvents := []events.SecretRotationEvent{
		{
			SecretIdentifier:  "aws://secret/arn:aws:secretsmanager:us-east-1:123456789012:secret:mysecret",
			RotationTimestamp: "2024-09-19T12:00:00Z",
			TriggerSource:     "aws-secretsmanager",
		},
	}
	return NewMockListener(mockEvents, time.Duration(source.Mock.EmitInterval)*time.Millisecond, eventChan), nil
}

func init() {
	schema.RegisterProvider(schema.Mock, &Provider{})
}
