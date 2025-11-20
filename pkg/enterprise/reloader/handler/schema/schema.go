/*
copyright External Secrets Inc. All Rights Reserved.
*/

// Package schema defines the handler schema and provider registry.
package schema

import (
	"context"
	"sync"

	"github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ExternalSecret is the ExternalSecret handler type.
	ExternalSecret = "ExternalSecret"
	// PushSecret is the PushSecret handler type.
	PushSecret = "PushSecret"
	// Deployment is the Deployment handler type.
	Deployment = "Deployment"
	// Workflow is the WorkflowRunTemplate handler type.
	Workflow = "WorkflowRunTemplate"
)

// ApplyFn is a function type for applying changes to an object.
type ApplyFn func(obj client.Object, event events.SecretRotationEvent) error

// ReferenceFn is a function type for checking if an object references a secret.
type ReferenceFn func(obj client.Object, secretName string) (bool, error)

// WaitForFn is a function type for waiting for an object to be ready.
type WaitForFn func(obj client.Object) error

// Handler defines the interface for handling secret rotation events.
type Handler interface {
	// Method to implement References
	// In the future, `matchStrategy` will just replace the References Method
	References(obj client.Object, secretName string) (bool, error)

	// Method to implement Apply
	// In the future, `updateStrategy` will create a new Apply method
	Apply(obj client.Object, event events.SecretRotationEvent) error

	// Method to implement WaitFor
	// In the future, `waitStrategy` will create a new WaitFor method
	WaitFor(obj client.Object) error

	// Filter implements the filter logic given the selected destination
	// Returns all objects that match the specific destination configuration
	Filter(destination *v1alpha1.DestinationToWatch, event events.SecretRotationEvent) ([]client.Object, error)
	WithApply(fn ApplyFn) Handler
	WithReference(fn ReferenceFn) Handler
	WithWaitFor(fn WaitForFn) Handler
}

// Provider is an interface for creating handlers.
type Provider interface {
	NewHandler(ctx context.Context, client client.Client, destination v1alpha1.DestinationToWatch) Handler
}

var (
	providerMap sync.Map
)

func init() {
	providerMap = sync.Map{}
}

// ForceRegister forcefully registers a provider, overwriting any existing provider.
func ForceRegister(name string, prov Provider) {
	providerMap.Store(name, prov)
}

// RegisterProvider registers a provider if it doesn't already exist.
func RegisterProvider(name string, prov Provider) bool {
	actual, loaded := providerMap.LoadOrStore(name, prov)
	if actual != nil {
		return false
	}
	return loaded
}
// GetProvider retrieves a registered provider by name.
func GetProvider(name string) Provider {
	content, exists := providerMap.Load(name)
	if !exists {
		return nil
	}
	h, ok := content.(Provider)
	if !ok {
		return nil
	}
	return h
}
