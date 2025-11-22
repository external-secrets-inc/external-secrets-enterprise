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

// AzureEventGridConfig contains configuration for Azure Event Grid notifications.
type AzureEventGridConfig struct {
	Host string `json:"host"`

	// +required
	// +kubebuilder:default=8080
	Port int32 `json:"port"`

	Subscriptions []string `json:"subscriptions"`
}
