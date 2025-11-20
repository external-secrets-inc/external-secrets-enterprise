// Copyright External Secrets Inc. 2025
// All Rights Reserved

// Package schema defines the listener schema and provider registry.
package schema

import (
	"context"
	"sync"

	"github.com/external-secrets/external-secrets/apis/enterprise/reloader/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/enterprise/reloader/events"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// AWSSQS is the AWS SQS listener type.
	AWSSQS          = "AwsSqs"
	// AzureEventGrid is the Azure Event Grid listener type.
	AzureEventGrid = "AzureEventGrid"
	// GooglePubSub is the Google Pub/Sub listener type.
	GooglePubSub = "GooglePubSub"
	// Webhook is the Webhook listener type.
	Webhook = "Webhook"
	// TCPSocket is the TCP Socket listener type.
	TCPSocket = "TCPSocket"
	// HashicorpVault is the Hashicorp Vault listener type.
	HashicorpVault = "HashicorpVault"
	// Mock is the Mock listener type.
	Mock = "Mock"
	// KubernetesSecret is the Kubernetes Secret listener type.
	KubernetesSecret = "KubernetesSecret"
	// KubernetesConfigMap is the Kubernetes ConfigMap listener type.
	KubernetesConfigMap = "KubernetesConfigMap"
)

var (
	providers sync.Map // map[string]Listener
)

func init() {
	providers = sync.Map{}
}

// Listener defines the interface for starting and stopping a listener.
type Listener interface {
	Start() error
	Stop() error
}

// Provider is an interface for creating event listeners for secret rotation events.
type Provider interface {
	CreateListener(ctx context.Context, source *v1alpha1.NotificationSource, client client.Client, eventChan chan events.SecretRotationEvent, logger logr.Logger) (Listener, error)
}

// ForceRegister forcefully registers a provider, overwriting any existing provider.
func ForceRegister(name string, prov Provider) {
	providers.Store(name, prov)
}

// RegisterProvider registers a provider if it doesn't already exist.
func RegisterProvider(name string, prov Provider) bool {
	if _, loaded := providers.LoadOrStore(name, prov); loaded {
		return false
	}
	return true
}

// GetProvider retrieves a registered provider by name.
func GetProvider(name string) Provider {
	if prov, loaded := providers.Load(name); loaded {
		if p, ok := prov.(Provider); ok {
			return p
		}
	}
	return nil
}
