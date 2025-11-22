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

// Package vault defines Vault-related models.
package vault

import "time"

// ValidMessage checks if an audit log message is valid.
func ValidMessage(m *AuditLog) bool {
	return m.AuthType == "response" && m.AuthResponse != nil && m.AuthResponse.MountType == "kv"
}

// AuditLog represents a Vault audit log entry.
type AuditLog struct {
	Auth         *Auth         `json:"auth,omitempty"`
	AuthRequest  *AuthRequest  `json:"request,omitempty"`
	AuthResponse *AuthResponse `json:"response,omitempty"`
	Time         time.Time     `json:"time,omitempty"`
	AuthType     string        `json:"type,omitempty"`
}

// Auth contains Vault authentication information.
type Auth struct {
	Accessor       string         `json:"accessor,omitempty"`
	DisplayName    string         `json:"display_name,omitempty"`
	Policies       []string       `json:"policies,omitempty"`
	PolicyResults  *PolicyResults `json:"policy_results,omitempty"`
	ClientToken    string         `json:"client_token,omitempty"`
	TokenPolicies  []string       `json:"token_policies,omitempty"`
	TokenIssueTime time.Time      `json:"token_issue_time,omitempty"`
	TokenType      string         `json:"token_type,omitempty"`
}

// PolicyResults contains policy evaluation results.
type PolicyResults struct {
	Allowed          bool             `json:"allowed,omitempty"`
	GrantingPolicies []GrantingPolicy `json:"granting_policies,omitempty"`
}

// GrantingPolicy represents a policy that granted access.
type GrantingPolicy struct {
	Name        string `json:"name,omitempty"`
	NamespaceID string `json:"namespace_id,omitempty"`
	Type        string `json:"type,omitempty"`
}

// AuthRequest represents a Vault authentication request.
type AuthRequest struct {
	ClientID            string                 `json:"client_id,omitempty"`
	ClientToken         string                 `json:"client_token,omitempty"`
	ClientTokenAccessor string                 `json:"client_token_accessor,omitempty"`
	Data                map[string]interface{} `json:"data,omitempty"`
	ID                  string                 `json:"id,omitempty"`
	MountAccessor       string                 `json:"mount_accessor,omitempty"`
	MountClass          string                 `json:"mount_class,omitempty"`
	MountPoint          string                 `json:"mount_point,omitempty"`
	MountRunningVersion string                 `json:"mount_running_version,omitempty"`
	MountType           string                 `json:"mount_type,omitempty"`
	Namespace           *Namespace             `json:"namespace,omitempty"`
	Operation           string                 `json:"operation,omitempty"`
	Path                string                 `json:"path,omitempty"`
	RemoteAddress       string                 `json:"remote_address,omitempty"`
	RemotePort          int                    `json:"remote_port,omitempty"`
	RequestURI          string                 `json:"request_uri,omitempty"`
}

// Namespace represents a Vault namespace.
type Namespace struct {
	ID string `json:"id,omitempty"`
}

// AuthResponse represents a Vault authentication response.
type AuthResponse struct {
	Data                      map[string]interface{} `json:"data,omitempty"`
	MountAccessor             string                 `json:"mount_accessor,omitempty"`
	MountClass                string                 `json:"mount_class,omitempty"`
	MountPoint                string                 `json:"mount_point,omitempty"`
	MountRunningPluginVersion string                 `json:"mount_running_plugin_version,omitempty"`
	MountType                 string                 `json:"mount_type,omitempty"`
}
