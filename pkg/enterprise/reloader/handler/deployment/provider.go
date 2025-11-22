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

// Package deployment implements deployment handler.
package deployment

import (
	"context"

	"github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider implements the deployment handler provider.
type Provider struct{}

// NewHandler creates a new deployment handler.
func (p *Provider) NewHandler(ctx context.Context, client client.Client, cache v1alpha1.DestinationToWatch) schema.Handler {
	h := &Handler{
		ctx:              ctx,
		client:           client,
		destinationCache: cache,
	}
	h.applyFn = h._apply
	h.referenceFn = h._references
	h.waitForFn = h._waitFor
	return h
}

func init() {
	schema.RegisterProvider(schema.Deployment, &Provider{})
}
