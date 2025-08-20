// Copyright External Secrets Inc. 2025
// All Rights reserved.
package virtualmachine

import tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"

type Request struct {
	Regexes   []string `json:"regexes"`
	Threshold int      `json:"threshold"`
	Paths     []string `json:"paths"`
}

type ScanResponse struct {
	JobId string `json:"jobId"`
}

type ScanJobResponse struct {
	JobID      string  `json:"jobId"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
	FinishedAt string  `json:"finishedAt"`
	Match      []Match `json:"match"`
}

type Match struct {
	Key      string `json:"key"`
	Property string `json:"property"`
}

type PushRequest struct {
	Value string `json:"value"`
}

type ConsumerRequest struct {
	Location tgtv1alpha1.SecretInStoreRef `json:"location"`
	Paths    []string                     `json:"paths"`
}

type ConsumerScanJobResponse struct {
	Status    string              `json:"status"`
	Consumers []map[string]string `json:"consumers"`
}
