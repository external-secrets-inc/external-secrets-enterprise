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
