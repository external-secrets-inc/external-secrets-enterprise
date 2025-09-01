// Copyright External Secrets Inc. 2025
//
//	All Rights Reserved
package feature

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Mock Kubernetes object for testing.
type mockObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func (m *mockObject) DeepCopyObject() runtime.Object {
	return &mockObject{
		TypeMeta:   m.TypeMeta,
		ObjectMeta: m.ObjectMeta,
	}
}

func (m *mockObject) GetObjectKind() schema.ObjectKind {
	return &mockObject{
		TypeMeta: m.TypeMeta,
	}
}

func (m *mockObject) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   m.APIVersion,
		Version: "v1",
		Kind:    m.Kind,
	}
}

func (m *mockObject) SetGroupVersionKind(gvk schema.GroupVersionKind) {
	m.APIVersion = gvk.GroupVersion().String()
	m.Kind = gvk.Kind
}

// Basic Feature Operations Tests

func TestNewFeature(t *testing.T) {
	name := "test.feature"
	description := "A test feature for unit testing"

	feature := NewFeature(name, description)

	assert.NotNil(t, feature)
	assert.Equal(t, name, feature.Name())
	assert.Equal(t, description, feature.Description())
	assert.False(t, feature.IsAvailable()) // Should start disabled
}

func TestFeature_Name_Description(t *testing.T) {
	name := "test.name.feature"
	description := "Test description for feature functionality"

	feature := NewFeature(name, description)

	assert.Equal(t, name, feature.Name())
	assert.Equal(t, description, feature.Description())
}

func TestFeature_Enable_Disable(t *testing.T) {
	feature := NewFeature("test.enable.feature", "Test enable/disable")

	// Initially disabled
	assert.False(t, feature.IsAvailable())

	// Enable the feature
	err := feature.Enable()
	assert.NoError(t, err)
	assert.True(t, feature.IsAvailable())

	// Disable the feature
	err = feature.Disable()
	assert.NoError(t, err)
	assert.False(t, feature.IsAvailable())
}

func TestFeature_IsAvailable(t *testing.T) {
	feature := NewFeature("test.available.feature", "Test availability")

	// Test initial state
	assert.False(t, feature.IsAvailable())

	// Test after enabling
	feature.Enable()
	assert.True(t, feature.IsAvailable())

	// Test after disabling
	feature.Disable()
	assert.False(t, feature.IsAvailable())
}

// Object Registration Tests

func TestFeature_Register_Success(t *testing.T) {
	feature := NewFeature("test.register.feature", "Test registration")

	obj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-object",
			Namespace: "test-namespace",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestObject",
			APIVersion: "v1",
		},
	}

	err := feature.Register(obj)
	assert.NoError(t, err)
	assert.True(t, feature.IsRegistered(obj))
}

func TestFeature_Register_MaxLimitReached(t *testing.T) {
	feature := NewFeature("test.limit.feature", "Test max limit")

	// Set a limit of 2
	feature.SetMaxLimit(2)

	obj1 := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "obj1", Namespace: "ns1"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}
	obj2 := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "obj2", Namespace: "ns1"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}
	obj3 := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "obj3", Namespace: "ns1"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	// Register the first two objects successfully
	err := feature.Register(obj1)
	assert.NoError(t, err)
	err = feature.Register(obj2)
	assert.NoError(t, err)

	// The third object should fail due to limit
	err = feature.Register(obj3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max limit reached")
	assert.False(t, feature.IsRegistered(obj3))
}

func TestFeature_Register_UnlimitedObjects(t *testing.T) {
	feature := NewFeature("test.unlimited.feature", "Test unlimited registration")

	// Default maxLimit is 0, which means unlimited
	for i := 0; i < 10; i++ {
		obj := &mockObject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "obj" + string(rune(i)),
				Namespace: "test-ns",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "TestObject",
				APIVersion: "v1",
			},
		}

		err := feature.Register(obj)
		assert.NoError(t, err)
		assert.True(t, feature.IsRegistered(obj))
	}
}

func TestFeature_Unregister_Success(t *testing.T) {
	feature := NewFeature("test.unregister.feature", "Test unregistration")

	obj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test-obj", Namespace: "test-ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	// Register first
	err := feature.Register(obj)
	assert.NoError(t, err)
	assert.True(t, feature.IsRegistered(obj))

	// Then unregister
	err = feature.Unregister(obj)
	assert.NoError(t, err)
	assert.False(t, feature.IsRegistered(obj))
}

func TestFeature_IsRegistered(t *testing.T) {
	feature := NewFeature("test.isregistered.feature", "Test registration check")

	obj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "test-obj", Namespace: "test-ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	// Initially not registered
	assert.False(t, feature.IsRegistered(obj))

	// After registration
	feature.Register(obj)
	assert.True(t, feature.IsRegistered(obj))

	// After unregistration
	feature.Unregister(obj)
	assert.False(t, feature.IsRegistered(obj))
}

// Limit Enforcement Tests

func TestFeature_SetMaxLimit_Zero_Unlimited(t *testing.T) {
	feature := NewFeature("test.zero.limit.feature", "Test zero limit")

	feature.SetMaxLimit(0)

	// Should allow unlimited registrations
	for i := 0; i < 100; i++ {
		obj := &mockObject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "obj" + string(rune(i)),
				Namespace: "test-ns",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "TestObject",
				APIVersion: "v1",
			},
		}

		err := feature.Register(obj)
		assert.NoError(t, err)
	}
}

func TestFeature_SetMaxLimit_Positive_Limited(t *testing.T) {
	feature := NewFeature("test.positive.limit.feature", "Test positive limit")

	limit := 3
	feature.SetMaxLimit(limit)

	// Register up to the limit
	for i := 0; i < limit; i++ {
		obj := &mockObject{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "obj" + string(rune(i)),
				Namespace: "test-ns",
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "TestObject",
				APIVersion: "v1",
			},
		}

		err := feature.Register(obj)
		assert.NoError(t, err)
	}

	// Next registration should fail
	extraObj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "extra-obj", Namespace: "test-ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	err := feature.Register(extraObj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max limit reached")
}

func TestFeature_Register_BeyondLimit_Fails(t *testing.T) {
	feature := NewFeature("test.beyond.limit.feature", "Test beyond limit failure")

	feature.SetMaxLimit(1)

	obj1 := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "obj1", Namespace: "ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}
	obj2 := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "obj2", Namespace: "ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	// The first registration succeeds
	err := feature.Register(obj1)
	assert.NoError(t, err)

	// Second registration fails
	err = feature.Register(obj2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max limit reached")

	// First object still registered, second is not
	assert.True(t, feature.IsRegistered(obj1))
	assert.False(t, feature.IsRegistered(obj2))
}

// Expiry Date Handling Tests

func TestFeature_SetExpiryDate_ValidDate(t *testing.T) {
	feature := NewFeature("test.expiry.feature", "Test expiry date")

	validDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	feature.SetExpiryDate(validDate)

	// Access the implementation to check the expiry date was set correctly
	impl, ok := feature.(*FeatureImpl)
	require.True(t, ok)

	expectedTime, err := time.Parse(time.RFC3339, validDate)
	require.NoError(t, err)

	assert.True(t, impl.expiry.Equal(expectedTime))
}

func TestFeature_SetExpiryDate_InvalidDate_DefaultsToZero(t *testing.T) {
	feature := NewFeature("test.invalid.expiry.feature", "Test invalid expiry date")

	invalidDate := "not-a-valid-date"
	feature.SetExpiryDate(invalidDate)

	// Access the implementation to check the expiry date was set to zero
	impl, ok := feature.(*FeatureImpl)
	require.True(t, ok)

	assert.True(t, impl.expiry.IsZero())
}

// Helper Function Tests

func TestRegisterOrFail_NewObject(t *testing.T) {
	feature := NewFeature("test.helper.new.feature", "Test helper new object")

	obj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "new-obj", Namespace: "ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	err := RegisterOrFail(feature, obj)
	assert.NoError(t, err)
	assert.True(t, feature.IsRegistered(obj))
}

func TestRegisterOrFail_ExistingObject(t *testing.T) {
	feature := NewFeature("test.helper.existing.feature", "Test helper existing object")

	obj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "existing-obj", Namespace: "ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	// Register first
	feature.Register(obj)

	// RegisterOrFail should not fail for existing object
	err := RegisterOrFail(feature, obj)
	assert.NoError(t, err)
	assert.True(t, feature.IsRegistered(obj))
}

func TestRegisterOrFail_LimitReached(t *testing.T) {
	feature := NewFeature("test.helper.limit.feature", "Test helper limit reached")
	feature.SetMaxLimit(1)

	obj1 := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "obj1", Namespace: "ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}
	obj2 := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "obj2", Namespace: "ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	// Register first object
	feature.Register(obj1)

	// RegisterOrFail should fail for second object due to limit
	err := RegisterOrFail(feature, obj2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max limit reached")
}

func TestUnregisterIfNotFound_ObjectNotFound(t *testing.T) {
	feature := NewFeature("test.helper.notfound.feature", "Test helper object not found")

	obj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "not-found-obj", Namespace: "ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	// Simulate a "not found" error - in Kubernetes, this would be an IsNotFound error
	notFoundErr := errors.NewNotFound(schema.GroupResource{
		Group:    "test",
		Resource: "testobjects",
	}, "not-found-obj")

	result, err := UnregisterIfNotFound(feature, obj, notFoundErr)

	// Should return no error and empty result
	assert.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestUnregisterIfNotFound_OtherError(t *testing.T) {
	feature := NewFeature("test.helper.othererror.feature", "Test helper other error")

	obj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{Name: "error-obj", Namespace: "ns"},
		TypeMeta:   metav1.TypeMeta{Kind: "TestObject", APIVersion: "v1"},
	}

	// Simulate some other error (not "not found")
	otherErr := errors.NewInternalError(fmt.Errorf("internal server error"))

	result, err := UnregisterIfNotFound(feature, obj, otherErr)

	// Should return the original error
	assert.Error(t, err)
	assert.Equal(t, otherErr, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// Global Feature Registry Tests

func TestFeatureRegistry_Register(t *testing.T) {
	feature := NewFeature("test.registry.feature", "Test registry functionality")

	err := Register(feature)
	assert.NoError(t, err)

	// Should be able to retrieve it
	retrieved, ok := Get("test.registry.feature")
	assert.True(t, ok)
	assert.Equal(t, feature.Name(), retrieved.Name())

	// Clean up
	AvailableFeatures.Delete("test.registry.feature")
}

func TestFeatureRegistry_Register_Duplicate(t *testing.T) {
	feature1 := NewFeature("test.duplicate.feature", "First feature")
	feature2 := NewFeature("test.duplicate.feature", "Second feature")

	// First registration should succeed
	err := Register(feature1)
	assert.NoError(t, err)

	// Second registration should fail
	err = Register(feature2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Clean up
	AvailableFeatures.Delete("test.duplicate.feature")
}

func TestFeatureRegistry_ForceRegister(t *testing.T) {
	feature1 := NewFeature("test.force.feature", "First feature")
	feature2 := NewFeature("test.force.feature", "Second feature")

	// Register first feature
	Register(feature1)

	// Force register should overwrite without error
	ForceRegister(feature2)

	// Should get the second feature
	retrieved, ok := Get("test.force.feature")
	assert.True(t, ok)
	assert.Equal(t, "Second feature", retrieved.Description())

	// Clean up
	AvailableFeatures.Delete("test.force.feature")
}

func TestFeatureRegistry_Get_NotFound(t *testing.T) {
	_, ok := Get("non.existent.feature")
	assert.False(t, ok)
}

// ObjectId Generation Tests

func TestObjectId_Generation(t *testing.T) {
	feature := NewFeature("test.objectid.feature", "Test ObjectId generation")
	impl := feature.(*FeatureImpl)

	obj := &mockObject{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-object",
			Namespace: "test-namespace",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestObject",
			APIVersion: "apps/v1",
		},
	}

	objId := impl.objectId(obj)

	assert.Equal(t, "test-object", objId.Name)
	assert.Equal(t, "test-namespace", objId.Namespace)
	assert.Equal(t, "TestObject", objId.Kind)
	assert.Equal(t, "apps/v1", objId.Group)
	assert.Equal(t, "v1", objId.Version)
}
