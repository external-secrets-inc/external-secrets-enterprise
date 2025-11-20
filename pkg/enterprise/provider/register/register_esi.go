// Copyright External Secrets Inc. 2025
// All rights reserved.

// Package register registers enterprise providers.
package register

import (
	// Register externalsecrets provider.
	_ "github.com/external-secrets/external-secrets/pkg/enterprise/provider/externalsecrets"
)
