// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var VirtualMachineKind = "VirtualMachine"

type VirtualMachineSpec struct {
	URL      string          `json:"url"`
	Paths    []string        `json:"paths"`
	CABundle string          `json:"caBundle,omitempty"`
	Auth     *Authentication `json:"auth,omitempty"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type Authentication struct {
	Basic       *BasicAuth       `json:"basic,omitempty"`
	Certificate *CertificateAuth `json:"certificate,omitempty"`
}

type CertificateAuth struct {
	ClientCertificateSecretRef *esmeta.SecretKeySelector `json:"clientCertificateSecretRef"`
	ClientKeySecretRef         *esmeta.SecretKeySelector `json:"clientKeySecretRef"`
}
type BasicAuth struct {
	UsernameSecretRef *esmeta.SecretKeySelector `json:"usernameSecretRef"`
	PasswordSecretRef *esmeta.SecretKeySelector `json:"passwordSecretRef"`
}

// VirtualMachine is the schema to scan target virtual machines
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-target}
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineSpec `json:"spec,omitempty"`
}

// VirtualMachineList contains a list of VirtualMachine resources.
// +kubebuilder:object:root=true
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func (c *VirtualMachine) GetObjectMeta() *metav1.ObjectMeta {
	return &c.ObjectMeta
}

func (c *VirtualMachine) GetTypeMeta() *metav1.TypeMeta {
	return &c.TypeMeta
}

func (c *VirtualMachine) GetSpec() *esv1.SecretStoreSpec {
	return &esv1.SecretStoreSpec{}
}

func (c *VirtualMachine) GetStatus() esv1.SecretStoreStatus {
	return esv1.SecretStoreStatus{}
}

func (c *VirtualMachine) SetStatus(_ esv1.SecretStoreStatus) {
}

func (c *VirtualMachine) GetNamespacedName() string {
	return fmt.Sprintf("%s/%s", c.Namespace, c.Name)
}

func (c *VirtualMachine) GetKind() string {
	return VirtualMachineKind
}

func (c *VirtualMachine) Copy() esv1.GenericStore {
	return c.DeepCopy()
}

func init() {
	RegisterObjKind(VirtualMachineKind, &VirtualMachine{})
}
