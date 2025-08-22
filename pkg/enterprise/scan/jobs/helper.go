// Copyright External Secrets Inc. 2025
// All Rights Reserved

package job

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
)

func Sanitize(ref tgtv1alpha1.SecretInStoreRef) string {
	cleanedName := strings.ToLower(strings.TrimSpace(ref.Name))
	cleanedKind := strings.ToLower(strings.TrimSpace(ref.Kind))
	cleanedKey := strings.TrimSuffix(strings.TrimPrefix(ref.RemoteRef.Key, "/"), "/")
	ans := cleanedKind + "." + cleanedName + "." + cleanedKey
	if ref.RemoteRef.Property != "" {
		cleanedProperty := strings.TrimSuffix(strings.TrimPrefix(ref.RemoteRef.Property, "/"), "/")
		ans += "." + cleanedProperty
	}
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(ans, "_", "-"), "/", "-"), ":", "-"))
}

func EqualLocations(a, b tgtv1alpha1.SecretInStoreRef) bool {
	return a.Name == b.Name && a.Kind == b.Kind && a.APIVersion == b.APIVersion && a.RemoteRef.Key == b.RemoteRef.Key && a.RemoteRef.Property == b.RemoteRef.Property
}

func CompareLocations(a, b tgtv1alpha1.SecretInStoreRef) int {
	aIdx := fmt.Sprintf("%s.%s", a.RemoteRef.Key, a.RemoteRef.Property)
	if a.RemoteRef.Property == "" {
		aIdx = a.RemoteRef.Key
	}
	bIdx := fmt.Sprintf("%s.%s", b.RemoteRef.Key, b.RemoteRef.Property)
	if b.RemoteRef.Property == "" {
		bIdx = b.RemoteRef.Key
	}
	return strings.Compare(aIdx, bIdx)
}

func SortLocations(loc []tgtv1alpha1.SecretInStoreRef) {
	slices.SortFunc(loc, CompareLocations)
}

func FillAttributes(consumer *consumerAccum, kind string, attrs map[string]string) {
	switch kind {
	case tgtv1alpha1.VirtualMachineKind:
		consumer.spec.VMProcess = &v1alpha1.VMProcessSpec{
			Hostname:   attrs["hostname"],
			Executable: attrs["executable"],
			User:       attrs["user"],
		}
		if pid, ok := attrs["pid"]; ok {
			if p, err := strconv.ParseInt(pid, 10, 64); err == nil {
				consumer.spec.VMProcess.PID = p
			}
		}
	case tgtv1alpha1.GithubTargetKind:
		consumer.spec.GitHubActor = &v1alpha1.GitHubActorSpec{
			Repository:    attrs["repository"],
			ActorType:     attrs["actorType"],
			ActorLogin:    attrs["actorLogin"],
			ActorID:       attrs["actorID"],
			Event:         attrs["event"],
			WorkflowRunID: attrs["workflowRunID"],
		}
	case tgtv1alpha1.KubernetesTargetKind:
		ws := &v1alpha1.K8sWorkloadSpec{
			ClusterName:     attrs["clusterName"],
			Namespace:       attrs["namespace"],
			WorkloadKind:    attrs["workloadKind"],
			WorkloadGroup:   attrs["workloadGroup"],
			WorkloadVersion: attrs["workloadVersion"],
			WorkloadName:    attrs["workloadName"],
			WorkloadUID:     attrs["workloadUID"],
			Controller:      attrs["controller"],
		}

		pods := make([]v1alpha1.K8sPodItem, 0)
		if podJson := attrs["pods"]; podJson != "" {
			_ = json.Unmarshal([]byte(podJson), &pods)
		}
		consumer.spec.K8sWorkload = ws
		consumer.status.Pods = pods
	}
}
