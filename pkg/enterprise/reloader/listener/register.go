// Copyright External Secrets Inc. 2025
// All Rights Reserved
package listener

import (
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/eventgrid"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/hashivault"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/k8ssecret"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/mock"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/pubsub"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/sqs"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/tcp"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/listener/webhook"
)
