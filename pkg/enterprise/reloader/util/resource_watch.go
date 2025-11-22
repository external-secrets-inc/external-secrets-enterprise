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

// Package util implements resource watch.
// Copyright External Secrets Inc. 2025
// All Rights Reserved
package util //nolint:revive,nolintlint

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MatchesAnyNamespaceSelector checks if the namespace labels match any of the provided namespace selectors.
func MatchesAnyNamespaceSelector(ctx context.Context, obj client.Object, namespaceSelectors []labels.Selector, client client.Client) (bool, error) {
	if len(namespaceSelectors) == 0 {
		// If no namespace selectors are provided, it's not a match
		return true, nil
	}

	// Get the namespace object
	var namespace corev1.Namespace
	if err := client.Get(ctx, types.NamespacedName{Name: obj.GetNamespace()}, &namespace); err != nil {
		return false, fmt.Errorf("failed to get namespace: %w", err)
	}

	// Check if the namespace labels match any of the selectors
	for _, nsSelector := range namespaceSelectors {
		if nsSelector.Matches(labels.Set(namespace.Labels)) {
			return true, nil
		}
	}
	return false, nil
}

// MatchesLabelSelectors checks if the secret's labels match the provided label selector.
func MatchesLabelSelectors(_ context.Context, obj client.Object, labelSelector labels.Selector, _ client.Client) (bool, error) {
	if labelSelector == nil {
		// If no label selector is provided, consider it a match.
		return true, nil
	}

	return labelSelector.Matches(labels.Set(obj.GetLabels())), nil
}

// IsNameInList checks if the secret's name is in the provided names set.
func IsNameInList(obj client.Object, nameSet map[string]struct{}) bool {
	if len(nameSet) == 0 {
		// If no names are specified, consider it a match.
		return true
	}

	_, exists := nameSet[obj.GetName()]
	return exists
}
