// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

import (
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var KubernetesTargetKind = "KubernetesCluster"

type KubernetesClusterSpec struct {
	// configures the Kubernetes server Address.
	// +optional
	Server KubernetesServer `json:"server,omitempty"`

	// Auth configures how secret-manager authenticates with a Kubernetes instance.
	// +optional
	Auth *KubernetesAuth `json:"auth,omitempty"`

	// A reference to a secret that contains the auth information.
	// +optional
	AuthRef *esmeta.SecretKeySelector `json:"authRef,omitempty"`

	// namespaces controls which namespaces are in scope during scans.
	// If both include and exclude are empty, all namespaces are included.
	// Include/exclude support glob-like patterns (implementation detail in provider).
	// +optional
	Namespaces *NamespacesMatcher `json:"namespaces,omitempty"`

	// selector filters workloads/pods by labels before scan evaluation.
	// If empty, all labeled/unlabeled workloads are considered.
	// +optional
	Selector *metav1.LabelSelector `json:"selector,omitempty"`

	// scan toggles specific binding paths to consider as "manifest bindings".
	// +optional
	Scan *KubernetesScanOptions `json:"scan,omitempty"`
}

type KubernetesServer struct {

	// configures the Kubernetes server Address.
	// +kubebuilder:default=kubernetes.default
	// +optional
	URL string `json:"url,omitempty"`

	// CABundle is a base64-encoded CA certificate
	// +optional
	CABundle []byte `json:"caBundle,omitempty"`

	// see: https://external-secrets.io/v0.4.1/spec/#external-secrets.io/v1alpha1.CAProvider
	// +optional
	CAProvider *esv1.CAProvider `json:"caProvider,omitempty"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type KubernetesAuth struct {
	// has both clientCert and clientKey as secretKeySelector
	// +optional
	Cert *CertAuth `json:"cert,omitempty"`

	// use static token to authenticate with
	// +optional
	Token *TokenAuth `json:"token,omitempty"`

	// points to a service account that should be used for authentication
	// +optional
	ServiceAccount *esmeta.ServiceAccountSelector `json:"serviceAccount,omitempty"`
}

type CertAuth struct {
	ClientCert esmeta.SecretKeySelector `json:"clientCert,omitempty"`
	ClientKey  esmeta.SecretKeySelector `json:"clientKey,omitempty"`
}

type TokenAuth struct {
	BearerToken esmeta.SecretKeySelector `json:"bearerToken,omitempty"`
}

// NamespacesMatcher selects namespaces by include/exclude pattern lists.
// Empty/omitted fields mean "no constraint".
type NamespacesMatcher struct {
	// +optional
	Include []string `json:"include,omitempty"`
	// +optional
	Exclude []string `json:"exclude,omitempty"`
}

// KubernetesScanOptions selects which manifest bindings are recognized.
// All fields default to true except IncludeImagePullSecrets (default false).
type KubernetesScanOptions struct {
	// IncludeImagePullSecrets: consider spec.imagePullSecrets and SA.imagePullSecrets.
	// +kubebuilder:default=false
	// +optional
	IncludeImagePullSecrets bool `json:"includeImagePullSecrets,omitempty"`

	// IncludeEnvFrom: consider containers[*].envFrom[].secretRef and initContainers[] equivalents.
	// +kubebuilder:default=true
	// +optional
	IncludeEnvFrom bool `json:"includeEnvFrom,omitempty"`

	// IncludeEnvSecretKeyRefs: consider containers[*].env[].valueFrom.secretKeyRef (and initContainers).
	// +kubebuilder:default=true
	// +optional
	IncludeEnvSecretKeyRefs bool `json:"includeEnvSecretKeyRefs,omitempty"`

	// IncludeVolumeSecrets: consider volumes[].secret.secretName.
	// +kubebuilder:default=true
	// +optional
	IncludeVolumeSecrets bool `json:"includeVolumeSecrets,omitempty"`
}

// KubernetesCluster is the schema for a Kubernetes target.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:metadata:labels="external-secrets.io/component=controller"
// +kubebuilder:resource:scope=Namespaced,categories={external-secrets,external-secrets-target}
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Capabilities",type=string,JSONPath=`.status.capabilities`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:subresource:status
type KubernetesCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KubernetesClusterSpec `json:"spec,omitempty"`
	Status            TargetStatus          `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KubernetesClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubernetesCluster `json:"items"`
}

func (c *KubernetesCluster) GetObjectMeta() *metav1.ObjectMeta {
	return &c.ObjectMeta
}

func (c *KubernetesCluster) GetTypeMeta() *metav1.TypeMeta {
	return &c.TypeMeta
}

func (c *KubernetesCluster) GetSpec() *esv1.SecretStoreSpec {
	return &esv1.SecretStoreSpec{}
}

func (c *KubernetesCluster) GetStatus() esv1.SecretStoreStatus {
	return *TargetToSecretStoreStatus(&c.Status)
}

func (c *KubernetesCluster) SetStatus(status esv1.SecretStoreStatus) {
	convertedStatus := SecretStoreToTargetStatus(&status)
	c.Status.Capabilities = convertedStatus.Capabilities
	c.Status.Conditions = convertedStatus.Conditions
}

func (c *KubernetesCluster) GetNamespacedName() string {
	return fmt.Sprintf("%s/%s", c.Namespace, c.Name)
}

func (c *KubernetesCluster) GetKind() string {
	return KubernetesTargetKind
}

func (c *KubernetesCluster) Copy() esv1.GenericStore {
	return c.DeepCopy()
}

func (c *KubernetesCluster) GetTargetStatus() TargetStatus {
	return c.Status
}

func (c *KubernetesCluster) SetTargetStatus(status TargetStatus) {
	c.Status = status
}

func (c *KubernetesCluster) CopyTarget() GenericTarget {
	return c.DeepCopy()
}

func init() {
	RegisterObjKind(KubernetesTargetKind, &KubernetesCluster{})
}
