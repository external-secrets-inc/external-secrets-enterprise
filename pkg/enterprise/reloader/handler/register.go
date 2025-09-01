/*
copyright External Secrets Inc. All Rights Reserved.
*/

package handler

import (
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/deployment"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/externalsecret"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/pushsecret"
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/reloader/handler/workflow"
)
