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

// Package v1alpha1 contains API Schema definitions for the reloader v1alpha1 API group
// Copyright External Secrets Inc. 2025
// All rights reserved
package v1alpha1

// ServiceAccountSelector represents a Kubernetes service account with a name and namespace for selection purposes.
type ServiceAccountSelector struct {

	// Name specifies the name of the service account to be selected.
	// +required
	Name string `json:"name"`

	// ServiceAccountSelector represents a Kubernetes service account with a name and namespace for selection purposes.
	// +required
	Namespace string `json:"namespace"`
	// Audience specifies the `aud` claim for the service account token
	// If the service account uses a well-known annotation for e.g. IRSA or GCP Workload Identity
	// then this audiences will be appended to the list
	// +optional
	Audiences []string `json:"audiences,omitempty"`
}

// SecretKeySelector is used to reference a specific secret within a Kubernetes namespace.
// It contains the name of the secret and the namespace where it resides.
type SecretKeySelector struct {

	// Name specifies the name of the referenced Kubernetes secret.
	// +required
	Name string `json:"name"`

	// Key specifies the key within the referenced Kubernetes secret.
	// +required
	Key string `json:"key"`

	// Namespace specifies the Kubernetes namespace where the referenced secret resides.
	// +required
	Namespace string `json:"namespace"`
}
