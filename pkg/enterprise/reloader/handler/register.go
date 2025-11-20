/*
copyright External Secrets Inc. All Rights Reserved.
*/

package handler

import (
	// Register deployment handler.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/deployment"
	// Register externalsecret handler.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/externalsecret"
	// Register pushsecret handler.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/pushsecret"
	// Register workflow handler.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/workflow"
)
