// Copyright External Secrets Inc. 2025
// All Rights reserved.
package virtualmachine

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
