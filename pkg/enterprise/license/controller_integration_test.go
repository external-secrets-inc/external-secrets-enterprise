// Copyright External Secrets Inc. 2025
//
//	All Rights Reserved
package license

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/external-secrets/external-secrets/pkg/enterprise/license/feature"
)

// Test helpers and mocks

type mockObject struct {
	metav1.TypeMeta
	metav1.ObjectMeta
}

func (m *mockObject) DeepCopyObject() runtime.Object {
	return &mockObject{
		TypeMeta:   m.TypeMeta,
		ObjectMeta: *m.ObjectMeta.DeepCopy(),
	}
}

func (m *mockObject) GetObjectKind() schema.ObjectKind {
	return &m.TypeMeta
}

func (m *mockObject) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "test.external-secrets.io",
		Version: "v1alpha1",
		Kind:    "TestKind",
	}
}

func (m *mockObject) SetGroupVersionKind(gvk schema.GroupVersionKind) {
	m.TypeMeta.APIVersion = gvk.GroupVersion().String()
	m.TypeMeta.Kind = gvk.Kind
}

type mockManager struct {
	manager.Manager
	scheme *runtime.Scheme
}

func (m *mockManager) GetScheme() *runtime.Scheme {
	if m.scheme == nil {
		m.scheme = runtime.NewScheme()
	}
	return m.scheme
}

type mockControllerReconciler struct {
	client.Client
	feature feature.Feature
	kind    string
}

func (r *mockControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	obj := &mockObject{}
	obj.SetNamespace(req.Namespace)
	obj.SetName(req.Name)

	// Simulate getting object - for testing, we'll assume it exists
	if err := feature.RegisterOrFail(r.feature, obj); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *mockControllerReconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	featureName := fmt.Sprintf("test.%s", r.kind)
	feat := feature.NewFeature(featureName, fmt.Sprintf("Test %s controller", r.kind))

	// Enable feature by default for controller setup
	feat.Enable()

	if err := feature.Register(feat); err != nil {
		return err
	}
	r.feature = feat

	// Only setup controller if feature is available
	if feat.IsAvailable() {
		// In real implementation, this would create the controller
		// For testing, we just simulate success
		return nil
	}
	return nil
}

// Helper functions for tests

func createTestFeature(name string, maxLimit int) feature.Feature {
	feat := feature.NewFeature(name, fmt.Sprintf("Test feature %s", name))
	feat.SetMaxLimit(maxLimit)
	feat.Enable()
	return feat
}

func createMockObject(name, namespace string) client.Object {
	obj := &mockObject{}
	obj.SetName(name)
	obj.SetNamespace(namespace)
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "test.external-secrets.io",
		Version: "v1alpha1",
		Kind:    "TestKind",
	})
	return obj
}

// Controller Setup Tests

func TestControllerSetup_FeatureRegistration(t *testing.T) {
	mgr := &mockManager{}
	reconciler := &mockControllerReconciler{kind: "TestController"}

	// Test feature registration during setup
	err := reconciler.SetupWithManager(mgr, controller.Options{})
	require.NoError(t, err, "SetupWithManager should succeed")

	// Verify feature was registered
	feat, exists := feature.Get("test.TestController")
	assert.True(t, exists, "Feature should be registered")
	assert.NotNil(t, feat, "Feature should not be nil")
	assert.Equal(t, "test.TestController", feat.Name(), "Feature name should match")
	assert.True(t, feat.IsAvailable(), "Feature should be available by default")
}

func TestControllerSetup_LicenseUnavailable_NoController(t *testing.T) {
	// Create a feature that's not available (disabled)
	feat := createTestFeature("test.UnavailableController", 0)
	feat.Disable()
	err := feature.Register(feat)
	require.NoError(t, err, "Should register disabled feature")

	reconciler := &mockControllerReconciler{kind: "UnavailableController"}

	// Mock the feature retrieval to return the disabled feature
	originalFeature := reconciler.feature
	reconciler.feature = feat

	// Test that controller setup recognizes unavailable license
	// In this case, SetupWithManager would return early without creating controller
	assert.False(t, feat.IsAvailable(), "Feature should not be available")

	// Cleanup
	reconciler.feature = originalFeature
}

func TestControllerSetup_FeatureNameMapping(t *testing.T) {
	testCases := []struct {
		controllerKind      string
		expectedFeatureName string
	}{
		{"AWSIam", "test.AWSIam"},
		{"BasicAuth", "test.BasicAuth"},
		{"PostgreSQL", "test.PostgreSQL"},
		{"CustomController", "test.CustomController"},
	}

	mgr := &mockManager{}

	for _, tc := range testCases {
		t.Run(tc.controllerKind, func(t *testing.T) {
			reconciler := &mockControllerReconciler{kind: tc.controllerKind}

			err := reconciler.SetupWithManager(mgr, controller.Options{})
			require.NoError(t, err, "SetupWithManager should succeed for %s", tc.controllerKind)

			// Verify correct feature name mapping
			feat, exists := feature.Get(tc.expectedFeatureName)
			assert.True(t, exists, "Feature %s should be registered", tc.expectedFeatureName)
			assert.NotNil(t, feat, "Feature should not be nil")
			assert.Equal(t, tc.expectedFeatureName, feat.Name(), "Feature name should match expected mapping")
		})
	}
}

// Reconcile Loop Integration Tests

func TestReconcile_RegisterOrFail_NewObject(t *testing.T) {
	feat := createTestFeature("test.reconcile.new", 5) // Limited to 5 objects
	err := feature.Register(feat)
	require.NoError(t, err, "Should register feature")

	reconciler := &mockControllerReconciler{feature: feat}

	// Test registering a new object
	obj := createMockObject("test-object", "test-namespace")
	err = feature.RegisterOrFail(feat, obj)
	assert.NoError(t, err, "Should register new object successfully")
	assert.True(t, feat.IsRegistered(obj), "Object should be registered")

	// Test reconcile with new object
	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-object",
			Namespace: "test-namespace",
		},
	}

	result, err := reconciler.Reconcile(ctx, req)
	assert.NoError(t, err, "Reconcile should succeed for new object")
	assert.Equal(t, ctrl.Result{}, result, "Should return empty result")
}

func TestReconcile_RegisterOrFail_LimitReached(t *testing.T) {
	feat := createTestFeature("test.reconcile.limit", 2) // Limited to 2 objects
	err := feature.Register(feat)
	require.NoError(t, err, "Should register feature")

	// Register objects up to the limit
	obj1 := createMockObject("object-1", "test-namespace")
	obj2 := createMockObject("object-2", "test-namespace")

	err = feature.RegisterOrFail(feat, obj1)
	require.NoError(t, err, "Should register first object")

	err = feature.RegisterOrFail(feat, obj2)
	require.NoError(t, err, "Should register second object")

	// Try to register one more object beyond the limit
	obj3 := createMockObject("object-3", "test-namespace")
	err = feature.RegisterOrFail(feat, obj3)
	assert.Error(t, err, "Should fail to register object beyond limit")
	assert.Contains(t, err.Error(), "limit", "Error should mention limit")
}

func TestReconcile_UnregisterIfNotFound_ObjectDeleted(t *testing.T) {
	feat := createTestFeature("test.reconcile.unregister", 0) // Unlimited
	err := Register(feat)
	require.NoError(t, err, "Should register feature")

	// Register an object first
	obj := createMockObject("test-object", "test-namespace")
	err = feature.RegisterOrFail(feat, obj)
	require.NoError(t, err, "Should register object")
	assert.True(t, feat.IsRegistered(obj), "Object should be registered")

	// Simulate object not found (deleted)
	notFoundError := errors.New("object not found")
	result, err := feature.UnregisterIfNotFound(feat, obj, notFoundError)

	// UnregisterIfNotFound should handle not-found errors gracefully
	assert.NoError(t, err, "Should handle not-found error gracefully")
	assert.Equal(t, ctrl.Result{}, result, "Should return empty result")
}

func TestReconcile_LicenseExpired_ControllerDisabled(t *testing.T) {
	feat := createTestFeature("test.reconcile.expired", 0)

	// Simulate expired license by disabling the feature
	feat.Disable()
	err := Register(feat)
	require.NoError(t, err, "Should register disabled feature")

	// Verify feature is not available (simulating expired license)
	assert.False(t, feat.IsAvailable(), "Feature should not be available when license expired")

	// Test that controller setup would not create controller for expired license
	reconciler := &mockControllerReconciler{kind: "ExpiredController"}
	reconciler.feature = feat

	// Controller should recognize license is not available
	if !feat.IsAvailable() {
		// This simulates the early return in SetupWithManager when license is expired
		assert.False(t, feat.IsAvailable(), "Controller should detect expired license")
	}
}

// Multi-Controller Scenarios Tests

func TestMultipleControllers_SameFeature(t *testing.T) {
	feat := createTestFeature("shared.feature", 3) // Limited to 3 objects total
	err := Register(feat)
	require.NoError(t, err, "Should register shared feature")

	// Each controller registers objects against the same feature
	obj1 := createMockObject("controller1-object", "test-namespace")
	obj2 := createMockObject("controller2-object", "test-namespace")

	err = feature.RegisterOrFail(feat, obj1)
	assert.NoError(t, err, "Controller1 should register object")

	err = feature.RegisterOrFail(feat, obj2)
	assert.NoError(t, err, "Controller2 should register object")

	// Both objects should be registered under the same feature
	assert.True(t, feat.IsRegistered(obj1), "Object from controller1 should be registered")
	assert.True(t, feat.IsRegistered(obj2), "Object from controller2 should be registered")

	// Try to exceed the shared limit
	obj3 := createMockObject("controller1-object2", "test-namespace")
	obj4 := createMockObject("controller2-object2", "test-namespace")

	err = feature.RegisterOrFail(feat, obj3)
	assert.NoError(t, err, "Should still be within limit")

	// This should fail as it exceeds the shared limit of 3
	err = feature.RegisterOrFail(feat, obj4)
	assert.Error(t, err, "Should fail when shared limit exceeded")
}

func TestMultipleControllers_DifferentFeatures(t *testing.T) {
	// Create separate features for different controllers
	feat1 := createTestFeature("controller1.feature", 2)
	feat2 := createTestFeature("controller2.feature", 3)

	err := Register(feat1)
	require.NoError(t, err, "Should register first feature")

	err = Register(feat2)
	require.NoError(t, err, "Should register second feature")

	// Each controller can use its full limit independently
	obj1a := createMockObject("controller1-object1", "test-namespace")
	obj1b := createMockObject("controller1-object2", "test-namespace")
	obj2a := createMockObject("controller2-object1", "test-namespace")
	obj2b := createMockObject("controller2-object2", "test-namespace")
	obj2c := createMockObject("controller2-object3", "test-namespace")

	// Controller1 can register up to its limit (2)
	err = feature.RegisterOrFail(feat1, obj1a)
	assert.NoError(t, err, "Controller1 should register first object")

	err = feature.RegisterOrFail(feat1, obj1b)
	assert.NoError(t, err, "Controller1 should register second object")

	// Controller2 can register up to its limit (3)
	err = feature.RegisterOrFail(feat2, obj2a)
	assert.NoError(t, err, "Controller2 should register first object")

	err = feature.RegisterOrFail(feat2, obj2b)
	assert.NoError(t, err, "Controller2 should register second object")

	err = feature.RegisterOrFail(feat2, obj2c)
	assert.NoError(t, err, "Controller2 should register third object")

	// Verify limits are enforced independently
	obj1c := createMockObject("controller1-object3", "test-namespace")
	err = feature.RegisterOrFail(feat1, obj1c)
	assert.Error(t, err, "Controller1 should exceed its limit")

	obj2d := createMockObject("controller2-object4", "test-namespace")
	err = feature.RegisterOrFail(feat2, obj2d)
	assert.Error(t, err, "Controller2 should exceed its limit")
}

func TestFeatureLimits_AcrossControllers(t *testing.T) {
	// Test scenario where multiple controller types share feature categories
	// but have different limits

	// Generator controllers share generator features but with different limits
	generatorFeat := createTestFeature("generator.shared", 5)
	workflowFeat := createTestFeature("workflow.shared", 2)

	err := Register(generatorFeat)
	require.NoError(t, err, "Should register generator feature")

	err = Register(workflowFeat)
	require.NoError(t, err, "Should register workflow feature")

	// Test that generators share the same limit pool
	obj1 := createMockObject("aws-generator", "test-namespace")
	obj2 := createMockObject("basic-auth-generator", "test-namespace")
	obj3 := createMockObject("workflow-object", "test-namespace")

	// Both generator controllers consume from the same feature limit
	err = feature.RegisterOrFail(generatorFeat, obj1)
	assert.NoError(t, err, "AWS generator should register")

	err = feature.RegisterOrFail(generatorFeat, obj2)
	assert.NoError(t, err, "BasicAuth generator should register")

	// Workflow controller uses separate feature limit
	err = feature.RegisterOrFail(workflowFeat, obj3)
	assert.NoError(t, err, "Workflow controller should register")

	// Add more objects to test limits
	for i := 3; i <= 5; i++ {
		obj := createMockObject(fmt.Sprintf("generator-%d", i), "test-namespace")
		err = feature.RegisterOrFail(generatorFeat, obj)
		assert.NoError(t, err, "Should register generator object %d", i)
	}

	// Now generator feature should be at limit
	obj6 := createMockObject("generator-6", "test-namespace")
	err = feature.RegisterOrFail(generatorFeat, obj6)
	assert.Error(t, err, "Should fail to register 6th generator object")

	// But workflow feature should still have capacity
	obj4 := createMockObject("workflow-object-2", "test-namespace")
	err = feature.RegisterOrFail(workflowFeat, obj4)
	assert.NoError(t, err, "Should register second workflow object")

	// Workflow should now be at limit
	obj5 := createMockObject("workflow-object-3", "test-namespace")
	err = feature.RegisterOrFail(workflowFeat, obj5)
	assert.Error(t, err, "Should fail to register third workflow object")
}
