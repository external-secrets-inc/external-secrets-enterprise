// Copyright External Secrets Inc. 2025
// All rights reserved

package v1alpha1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
)

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil
type TargetProvider interface {
	NewClient(ctx context.Context, client client.Client, target client.Object) (ScanTarget, error)
}

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil
type ScanTarget interface {
	ScanForSecrets(ctx context.Context, regexes []string, threshold int) ([]scanv1alpha1.SecretInStoreRef, error)
	ScanForConsumers(ctx context.Context, location scanv1alpha1.SecretInStoreRef, hash string) ([]scanv1alpha1.ConsumerFinding, error)
	Lock()
	Unlock()
}

// +kubebuilder:object:root=false
// +kubebuilder:object:generate:false
// +k8s:deepcopy-gen:interfaces=nil
// +k8s:deepcopy-gen=nil
// GenericTarget is a common interface for interacting with Targets.
type GenericTarget interface {
	runtime.Object
	metav1.Object

	GetObjectMeta() *metav1.ObjectMeta
	GetTypeMeta() *metav1.TypeMeta
	GetKind() string

	GetNamespacedName() string
	GetTargetStatus() TargetStatus
	SetTargetStatus(status TargetStatus)
	CopyTarget() GenericTarget
}
