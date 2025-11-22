// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package v1alpha1 contains the virtual machine target types.
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualMachineKind is the kind name for VirtualMachine resources.
var VirtualMachineKind = "VirtualMachine"

// VirtualMachineSpec contains the virtual machine spec.
type VirtualMachineSpec struct {
	URL      string          `json:"url"`
	Paths    []string        `json:"paths"`
	CABundle string          `json:"caBundle,omitempty"`
	Auth     *Authentication `json:"auth,omitempty"`
}

// Authentication contains the authentication information for the virtual machine.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type Authentication struct {
	Basic       *BasicAuth       `json:"basic,omitempty"`
	Certificate *CertificateAuth `json:"certificate,omitempty"`
}

// CertificateAuth contains the client certificate and key for certificate authentication.
type CertificateAuth struct {
	ClientCertificateSecretRef *esmeta.SecretKeySelector `json:"clientCertificateSecretRef"`
	ClientKeySecretRef         *esmeta.SecretKeySelector `json:"clientKeySecretRef"`
}

// BasicAuth contains the username and password for basic authentication.
type BasicAuth struct {
	UsernameSecretRef *esmeta.SecretKeySelector `json:"usernameSecretRef"`
	PasswordSecretRef *esmeta.SecretKeySelector `json:"passwordSecretRef"`
}

// VirtualMachine is the schema to scan target virtual machines
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets, external-secrets-target}
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Capabilities",type=string,JSONPath=`.status.capabilities`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:subresource:status
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VirtualMachineSpec `json:"spec,omitempty"`
	Status            TargetStatus       `json:"status,omitempty"`
}

// VirtualMachineList contains a list of VirtualMachine resources.
// +kubebuilder:object:root=true
type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

// GetObjectMeta returns the object meta.
func (c *VirtualMachine) GetObjectMeta() *metav1.ObjectMeta {
	return &c.ObjectMeta
}

// GetTypeMeta returns the type meta.
func (c *VirtualMachine) GetTypeMeta() *metav1.TypeMeta {
	return &c.TypeMeta
}

// GetSpec returns the spec of the object.
func (c *VirtualMachine) GetSpec() *esv1.SecretStoreSpec {
	return &esv1.SecretStoreSpec{}
}

// GetStatus returns the status of the object.
func (c *VirtualMachine) GetStatus() esv1.SecretStoreStatus {
	return *TargetToSecretStoreStatus(&c.Status)
}

// SetStatus sets the status of the object.
func (c *VirtualMachine) SetStatus(status esv1.SecretStoreStatus) {
	convertedStatus := SecretStoreToTargetStatus(&status)
	c.Status.Capabilities = convertedStatus.Capabilities
	c.Status.Conditions = convertedStatus.Conditions
}

// GetNamespacedName returns the namespaced name of the object.
func (c *VirtualMachine) GetNamespacedName() string {
	return fmt.Sprintf("%s/%s", c.Namespace, c.Name)
}

// GetKind returns the kind of the object.
func (c *VirtualMachine) GetKind() string {
	return VirtualMachineKind
}

// Copy returns a copy of the object.
func (c *VirtualMachine) Copy() esv1.GenericStore {
	return c.DeepCopy()
}

// GetTargetStatus returns the target status.
func (c *VirtualMachine) GetTargetStatus() TargetStatus {
	return c.Status
}

// SetTargetStatus sets the target status.
func (c *VirtualMachine) SetTargetStatus(status TargetStatus) {
	c.Status = status
}

// CopyTarget returns a copy of the target.
func (c *VirtualMachine) CopyTarget() GenericTarget {
	return c.DeepCopy()
}

func init() {
	RegisterObjKind(VirtualMachineKind, &VirtualMachine{})
}
