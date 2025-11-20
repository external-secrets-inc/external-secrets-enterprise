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

// Package events provides event types for secret rotation.
// Copyright External Secrets Inc. 2025
// All Rights Reserved.
package events

// SecretRotationEvent represents an event triggered during the secret rotation process.
// It contains the secret identifier, the timestamp of the rotation, and the source that triggered the event.
type SecretRotationEvent struct {
	SecretIdentifier  string
	RotationTimestamp string
	TriggerSource     string
	// Optional bit so we can filter down better depending on the namespace.
	Namespace string
}
