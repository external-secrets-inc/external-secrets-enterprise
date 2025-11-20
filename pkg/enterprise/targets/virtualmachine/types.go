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
// All Rights reserved.

// Package virtualmachine implements virtual machine targets
package virtualmachine

import (
	"time"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
)

// Request represents a scan request.
type Request struct {
	Regexes   []string `json:"regexes"`
	Threshold int      `json:"threshold"`
	Paths     []string `json:"paths"`
}

// ScanResponse represents a scan response.
type ScanResponse struct {
	// JobID is the ID of the scan job.
	JobID string `json:"jobId"`
}

// ScanJobResponse represents a scan job response.
type ScanJobResponse struct {
	JobID      string  `json:"jobId"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
	FinishedAt string  `json:"finishedAt"`
	Match      []Match `json:"match"`
}

// Match represents a secret match.
type Match struct {
	Key      string `json:"key"`
	Property string `json:"property"`
}

// PushRequest represents a push request.
type PushRequest struct {
	Value string `json:"value"`
}

// ConsumerRequest represents a consumer scan request.
type ConsumerRequest struct {
	Location scanv1alpha1.SecretInStoreRef `json:"location"`
	Paths    []string                      `json:"paths"`
}

// ConsumerScanJobResponse represents a consumer scan job response.
type ConsumerScanJobResponse struct {
	ID        uint      `json:"id"`
	CreatedAt time.Time `json:"createdAt,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
	DeletedAt time.Time `json:"deletedAt,omitempty"`
	FilePath  string    `json:"filePath"`
	Comm      string    `json:"comm"`
	Exe       string    `json:"exe"`
	RUID      int       `json:"ruid"`
	EUID      int       `json:"euid"`
}
