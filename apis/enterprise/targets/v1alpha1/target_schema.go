// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	"fmt"
	"sync"
)

var builder map[string]TargetProvider
var buildlock sync.RWMutex

func init() {
	builder = make(map[string]TargetProvider)
}

// Register a generator type. Register panics if a
// backend with the same generator is already registered.
func Register(kind string, g TargetProvider) {
	buildlock.Lock()
	defer buildlock.Unlock()
	_, exists := builder[kind]
	if exists {
		panic(fmt.Sprintf("kind %q already registered", kind))
	}

	builder[kind] = g
}

// ForceRegister adds to the schema, overwriting a generator if
// already registered. Should only be used for testing.
func ForceRegister(kind string, g TargetProvider) {
	buildlock.Lock()
	builder[kind] = g
	buildlock.Unlock()
}

// GetTargetByName returns the provider implementation by name.
func GetTargetByName(kind string) (TargetProvider, bool) {
	buildlock.RLock()
	f, ok := builder[kind]
	buildlock.RUnlock()
	return f, ok
}
