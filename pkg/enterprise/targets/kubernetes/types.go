// Copyright External Secrets Inc. 2025
// All Rights Reserved

package kubernetes

type podItem struct {
	Name     string `json:"name"`
	UID      string `json:"uid,omitempty"`
	NodeName string `json:"nodeName,omitempty"`
	Phase    string `json:"phase,omitempty"`
	Ready    bool   `json:"ready"`
	Reason   string `json:"reason,omitempty"`
}

type workloadRef struct {
	Group, Version, Kind, Namespace, Name, UID string
}
