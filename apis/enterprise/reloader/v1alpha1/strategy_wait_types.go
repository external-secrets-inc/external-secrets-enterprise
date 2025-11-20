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

/*
copyright External Secrets Inc. All Rights Reserved.
*/

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// WaitStrategy defines a wait strategy.
type WaitStrategy struct {
	// Waits for a given time interval to reconcile the next object
	//+optional
	Time *metav1.Duration `json:"time,omitempty"`
	// Waits for a given status condition to be met
	//+optional
	Condition *WaitForCondition `json:"condition,omitempty"`
}

// WaitForCondition defines a condition to wait for.
type WaitForCondition struct {
	// Period to wait before each retry
	//+optional
	RetryTimeout *metav1.Duration `json:"retryTimeout,omitempty"`
	// Maximum retries to check for a condition
	//+optional
	MaxRetries *int32 `json:"maxRetries,omitempty"`
	// The name of the condition to wait for
	//+required
	Type string `json:"type"`
	// The status of the condition to wait for
	//+optional
	Status string `json:"status"`
	// Optional message to match
	//+optional
	Message string `json:"message,omitempty"`
	// Optional reason to match
	//+optional
	Reason string `json:"reason,omitempty"`
	// Only accept this condition after a given period from the transition time
	//+optional
	TransitionedAfter *metav1.Duration `json:"transitionedAfter,omitempty"`
	// Only accept this condition after a given period from the update time
	//+optional
	UpdatedAfter *metav1.Duration `json:"updatedAfter,omitempty"`
}
