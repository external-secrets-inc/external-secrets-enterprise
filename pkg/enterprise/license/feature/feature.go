package feature

import (
	"errors"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectId struct {
	Namespace string
	Name      string
	Group     string
	Version   string
	Kind      string
}
type FeatureImpl struct {
	maxLimit    int
	expiry      time.Time
	registry    map[ObjectId]client.Object
	enabled     bool
	name        string
	description string
}

func NewFeature(name string, description string) Feature {
	return &FeatureImpl{
		name:        name,
		description: description,
		registry:    make(map[ObjectId]client.Object),
	}
}

func (f *FeatureImpl) Name() string {
	return f.name
}

func (f *FeatureImpl) Description() string {
	return f.description
}

func (f *FeatureImpl) Register(content client.Object) error {
	if f.maxLimit > 0 && len(f.registry) >= f.maxLimit {
		return errors.New("max limit reached")
	}
	f.registry[f.objectId(content)] = content
	return nil
}

func (f *FeatureImpl) Unregister(content client.Object) error {
	delete(f.registry, f.objectId(content))
	return nil
}

func (f *FeatureImpl) IsRegistered(content client.Object) bool {
	_, ok := f.registry[f.objectId(content)]
	return ok
}

func (f *FeatureImpl) IsAvailable() bool {
	return f.enabled
}

func (f *FeatureImpl) Enable() error {
	f.enabled = true
	return nil
}

func (f *FeatureImpl) Disable() error {
	f.enabled = false
	return nil
}

func (f *FeatureImpl) SetMaxLimit(limit int) {
	f.maxLimit = limit
}

func (f *FeatureImpl) SetExpiryDate(expiryDate string) {
	expiry, err := time.Parse(time.RFC3339, expiryDate)
	if err != nil {
		f.expiry = time.Time{}
		return
	}
	f.expiry = expiry
}

func (f *FeatureImpl) objectId(content client.Object) ObjectId {
	return ObjectId{
		Namespace: content.GetNamespace(),
		Name:      content.GetName(),
		Group:     content.GetObjectKind().GroupVersionKind().Group,
		Version:   content.GetObjectKind().GroupVersionKind().Version,
		Kind:      content.GetObjectKind().GroupVersionKind().Kind,
	}
}

func UnregisterIfNotFound(feature Feature, obj client.Object, err error) (ctrl.Result, error) {
	if client.IgnoreNotFound(err) == nil {
		if err := feature.Unregister(obj); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, err
}

func RegisterOrFail(feature Feature, obj client.Object) error {
	if feature.IsRegistered(obj) {
		return nil
	}
	if err := feature.Register(obj); err != nil {
		return err
	}
	return nil
}
