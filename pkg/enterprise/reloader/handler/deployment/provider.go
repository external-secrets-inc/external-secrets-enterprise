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
