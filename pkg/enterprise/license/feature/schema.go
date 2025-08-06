package feature

import (
	"fmt"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Feature interface {
	Name() string
	Description() string
	Register(content client.Object) error
	Unregister(content client.Object) error
	IsRegistered(content client.Object) bool
	IsAvailable() bool
	Enable() error
	Disable() error
	SetMaxLimit(limit int)
	SetExpiryDate(expiryDate string)
}

// AvailableFeatures defines the available features in the system
var AvailableFeatures = sync.Map{}

func Register(feature Feature) error {
	if _, loaded := AvailableFeatures.Load(feature.Name()); loaded {
		return fmt.Errorf("feature %s is already registered", feature.Name())
	}
	AvailableFeatures.Store(feature.Name(), feature)
	return nil
}

func ForceRegister(feature Feature) {
	AvailableFeatures.Store(feature.Name(), feature)
}

func Get(featureName string) (Feature, bool) {
	feature, ok := AvailableFeatures.Load(featureName)
	return feature.(Feature), ok
}
