// Copyright External Secrets Inc. 2025
// All Rights Reserved

package kubernetes

type workloadRef struct {
	Group, Version, Kind, Namespace, Name, UID string
}
