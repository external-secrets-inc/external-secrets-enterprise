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

// Copyright External Secrets Inc. 2025
// All Rights Reserved

// Package gcp defines GCP-related models.
package gcp

// AuditLogMessage represents a GCP audit log message.
type AuditLogMessage struct {
	ProtoPayload     AuditLog `json:"protoPayload"`
	Resource         Resource `json:"resource"`
	Timestamp        string   `json:"timestamp"`
	ReiveceTimestamp string   `json:"receiveTimestamp"`
}
// AuditLog represents a GCP audit log entry.
type AuditLog struct {
	AuthenticationInfo AuthenticationInfo `json:"authenticationInfo"`
	MethodName         string             `json:"methodName"`
	RequestMetadata    RequestMetadata    `json:"requestMetadata"`
	ResourceName       string             `json:"resourceName"`
	ServiceName        string             `json:"serviceName"`
}

// AuthenticationInfo contains authentication information.
type AuthenticationInfo struct {
	PrincipalEmail string `json:"principalEmail"`
}
// RequestMetadata contains request metadata.
type RequestMetadata struct {
	CallerIP       string `json:"callerIp"`
	CallerSupplied string `json:"callerSuppliedUserAgent"`
}
// Resource represents a GCP resource.
type Resource struct {
	Labels map[string]string `json:"labels"`
	Type   string            `json:"type"`
}
