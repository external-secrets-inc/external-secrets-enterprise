/*
Copyright External Secrets Inc. All Rights Reserved
*/

package externalsecret

import (
	"context"

	"github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider implements the ExternalSecret handler provider.
type Provider struct{}

// NewHandler creates a new ExternalSecret handler.
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
	schema.RegisterProvider(schema.ExternalSecret, &Provider{})
}
