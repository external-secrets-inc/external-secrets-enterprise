// Copyright External Secrets Inc. 2025
// All rights reserved.

package enterprise

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

// ExternalSecretsProvider configures the External Secrets Enterprise provider.
type ExternalSecretsProvider struct {
	// URL For the External Secrets Enterprise Server.
	// +required
	Server ExternalSecretsServer `json:"server"`

	// Authentication parameters for External Secrets Enterprise
	// +required
	Auth ExternalSecretsAuth `json:"auth"`

	Target ExternalSecretsTarget `json:"target"`
}

// ExternalSecretsTarget specifies the target for External Secrets Enterprise operations.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type ExternalSecretsTarget struct {
	// Remote clusterSecretStore to connect. Eventually, support more fields
	ClusterSecretStoreName *string `json:"clusterSecretStoreName,omitempty"`
}

// ExternalSecretsServer defines the server configuration for External Secrets Enterprise.
type ExternalSecretsServer struct {
	// +optional
	CaRef *ExternalSecretsCARef `json:"caRef,omitempty"`
	// URL For the External Secrets Enterprise Server.
	URL string `json:"url,omitempty"`
}

// ExternalSecretsAuth defines authentication methods for External Secrets Enterprise.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type ExternalSecretsAuth struct {
	Kubernetes *ExternalSecretsKubernetesAuth `json:"kubernetes,omitempty"`
}

// ExternalSecretsKubernetesAuth defines Kubernetes-based authentication for External Secrets Enterprise.
type ExternalSecretsKubernetesAuth struct {
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
	CaCertRef         ExternalSecretsCARef          `json:"caCertRef,omitempty"`
}

// ExternalSecretsCARef defines a reference to a CA certificate.
type ExternalSecretsCARef struct {
	Bundle       []byte                    `json:"bundle,omitempty"`
	SecretRef    *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
	ConfigMapRef *esmeta.SecretKeySelector `json:"configMapRef,omitempty"`
}
