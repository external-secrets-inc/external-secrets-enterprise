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

// DeploymentDestination defines a DeploymentDestination. Behavior is a pod templates annotations patch.
// Default UpdateStrategy is pod template annotations patch to trigger a new rollout.
// Default MatchStrategy is matching secret-key with any of:
// * Equality against `spec.template.spec.containers[*].env[*].valueFrom.secretKeyRef.name`
// * Equality against `spec.template.spec.containers[*].envFrom.secretRef.name`
// Default WaitStrategy is to wait for the rollout to be completed with 3 minutes of grace period before
// moving to the next matched deployment.
type DeploymentDestination struct {
	// NamespaceSelectors selects namespaces based on labels.
	// The manifest must reside in a namespace that matches at least one of these selectors.
	// +optional
	NamespaceSelectors []metav1.LabelSelector `json:"namespaceSelectors,omitempty"`

	// LabelSelectors selects resources based on their labels.
	// The resource must satisfy all conditions defined in this selector.
	// Supports both matchLabels and matchExpressions for advanced filtering.
	// +optional
	LabelSelectors *metav1.LabelSelector `json:"labelSelectors,omitempty"`

	// Names specifies a list of resource names to watch.
	// The resource must have a name that matches one of these entries.
	// +optional
	Names []string `json:"names,omitempty"`
}
