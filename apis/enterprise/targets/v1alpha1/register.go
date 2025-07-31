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
	Group   = "target.external-secrets.io"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects.
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme
	registry      = map[string]esv1.GenericStore{}
)

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}

func GetObjFromKind(kind string) esv1.GenericStore {
	return registry[kind]
}

func RegisterObjKind(kind string, obj esv1.GenericStore) {
	registry[kind] = obj
}
