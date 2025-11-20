// Package v1alpha1 contains API Schema definitions for the targets v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	// Group is the group name used to register these objects.
	Group = "target.external-secrets.io"
	// Version is the version name used to register these objects.
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	// AddToScheme is used to add go types to the GroupVersionKind scheme.
	AddToScheme = SchemeBuilder.AddToScheme
	// registry is a map of all registered targets.
	registry = map[string]esv1.GenericStore{}
)

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
	SchemeBuilder.Register(&GithubRepository{}, &GithubRepositoryList{})
	SchemeBuilder.Register(&KubernetesCluster{}, &KubernetesClusterList{})
}

// GetObjFromKind returns a registered target by kind.
func GetObjFromKind(kind string) esv1.GenericStore {
	return registry[kind]
}

// RegisterObjKind registers a target by kind.
func RegisterObjKind(kind string, obj esv1.GenericStore) {
	registry[kind] = obj
}

// GetAllTargets returns a map of all registered targets.
func GetAllTargets() map[string]esv1.GenericStore {
	return registry
}
