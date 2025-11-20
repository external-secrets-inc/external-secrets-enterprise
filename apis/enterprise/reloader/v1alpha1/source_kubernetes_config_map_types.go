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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// KubernetesConfigMapConfig contains configuration for Kubernetes notifications.
type KubernetesConfigMapConfig struct {
	// Server URL
	// +required
	ServerURL string `json:"serverURL"`

	// How to authenticate with Kubernetes cluster. If not specified, the default config is used.
	// +optional
	Auth *KubernetesAuth `json:"auth,omitempty"`

	// LabelSelector can be used to identify and narrow down secrets for watching.
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
}
