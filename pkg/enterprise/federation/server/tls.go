// Package server implements the federation server.
// Copyright External Secrets Inc.
// All Rights Reserved.
package server

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
)

var (
	allowedMu  sync.RWMutex
	allowedIDs = map[string]struct{}{}
)

// AddTLSAllowedID adds a new allowed ID to the TLS store.
func AddTLSAllowedID(id string) {
	allowedMu.Lock()
	defer allowedMu.Unlock()
	allowedIDs[id] = struct{}{}
}

// RemoveTLSAllowedID removes an allowed ID from the TLS store.
func RemoveTLSAllowedID(id string) {
	allowedMu.Lock()
	defer allowedMu.Unlock()
	delete(allowedIDs, id)
}

// isAllowed checks if the given ID is allowed.
func isAllowed(id string) bool {
	allowedMu.RLock()
	defer allowedMu.RUnlock()
	_, ok := allowedIDs[id]
	return ok
}

// verifyConnection verifies the connection state.
func verifyConnection(cs tls.ConnectionState) error {
	if len(cs.PeerCertificates) == 0 {
		return fmt.Errorf("no certificates found")
	}
	leaf := cs.PeerCertificates[0]

	id, err := x509svid.IDFromCert(leaf)
	if err != nil {
		return fmt.Errorf("error extracting spiffe id: %w", err)
	}

	if !isAllowed(id.String()) {
		return fmt.Errorf("not authorized spiffe id: %s", id.String())
	}
	return nil
}
