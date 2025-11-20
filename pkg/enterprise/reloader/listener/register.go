// Copyright External Secrets Inc. 2025
// All Rights Reserved

// Package listener manages event listeners for secret rotation.
package listener

import (
	// Register eventgrid listener.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/eventgrid"
	// Register hashivault listener.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/hashivault"
	// Register k8ssecret listener.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/k8ssecret"
	// Register mock listener.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/mock"
	// Register pubsub listener.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/pubsub"
	// Register sqs listener.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/sqs"
	// Register tcp listener.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/tcp"
	// Register webhook listener.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/webhook"
)
