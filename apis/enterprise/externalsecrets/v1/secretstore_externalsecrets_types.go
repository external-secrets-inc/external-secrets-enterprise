// Copyright External Secrets Inc. 2025
// All rights reserved.

package enterprise

import (
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type ExternalSecretsProvider struct {
	// URL For the External Secrets Enterprise Server.
	// +required
	Server ExternalSecretsServer `json:"server"`

	// Authentication parameters for External Secrets Enterprise
	// +required
	Auth ExternalSecretsAuth `json:"auth"`

	Target ExternalSecretsTarget `json:"target"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type ExternalSecretsTarget struct {
	// Remote clusterSecretStore to connect. Eventually, support more fields
	ClusterSecretStoreName *string `json:"clusterSecretStoreName,omitempty"`
}

type ExternalSecretsServer struct {
	// +optional
	CaRef *ExternalSecretsCARef `json:"caRef,omitempty"`
	// URL For the External Secrets Enterprise Server.
	URL string `json:"url,omitempty"`
}

// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type ExternalSecretsAuth struct {
	Kubernetes *ExternalSecretsKubernetesAuth `json:"kubernetes,omitempty"`
}

type ExternalSecretsKubernetesAuth struct {
	ServiceAccountRef esmeta.ServiceAccountSelector `json:"serviceAccountRef,omitempty"`
	CaCertRef         ExternalSecretsCARef          `json:"caCertRef,omitempty"`
}

type ExternalSecretsCARef struct {
	Bundle       []byte                    `json:"bundle,omitempty"`
	SecretRef    *esmeta.SecretKeySelector `json:"secretRef,omitempty"`
	ConfigMapRef *esmeta.SecretKeySelector `json:"configMapRef,omitempty"`
}
