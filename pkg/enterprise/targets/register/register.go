// Copyright External Secrets Inc. 2025
// All Rights reserved.

// Package targets registers all target providers.
package targets

import (
	// Register GitHub target provider.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/targets/github"
	// Register Kubernetes target provider.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/targets/kubernetes"
	// Register VirtualMachine target provider.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/targets/virtualmachine"
)
