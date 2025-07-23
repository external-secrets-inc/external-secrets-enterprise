package v1alpha1

import (
	"fmt"
	"sync"
)

var (
	genericBuilder = make(map[string]GenericGenerator)
	genericLock    sync.RWMutex
)

func RegisterGeneric(kind string, genericGenerator GenericGenerator) {
	genericLock.Lock()
	defer genericLock.Unlock()

	if _, exists := genericBuilder[kind]; exists {
		panic(fmt.Sprintf("Generic generator %q already registered", kind))
	}
	genericBuilder[kind] = genericGenerator
}

// ForceRegister adds to the schema, overwriting a generator if
// already registered. Should only be used for testing.
func ForceRegisterGeneric(kind string, genericGenerator GenericGenerator) {
	genericLock.Lock()
	genericBuilder[kind] = genericGenerator
	genericLock.Unlock()
}

func GetGenericByKind(kind string) (GenericGenerator, bool) {
	genericLock.RLock()
	genericGenerator, ok := genericBuilder[kind]
	genericLock.RUnlock()
	return genericGenerator, ok
}
