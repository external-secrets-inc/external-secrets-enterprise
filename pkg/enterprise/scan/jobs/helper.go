// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package job

import (
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

func FillUnionFromAttributes(spec *v1alpha1.ConsumerSpec, kind string, attrs map[string]string) {
	switch kind {
	case tgtv1alpha1.VirtualMachineKind:
		spec.VMProcess = &v1alpha1.VMProcessSpec{
			Hostname:   attrs["hostname"],
			Executable: attrs["executable"],
			User:       attrs["user"],
		}
		if pid, ok := attrs["pid"]; ok {
			if p, err := strconv.ParseInt(pid, 10, 64); err == nil {
				spec.VMProcess.PID = p
			}
		}
	case tgtv1alpha1.GithubTargetKind:
		spec.GitHubActor = &v1alpha1.GitHubActorSpec{
			Repository:    attrs["repository"],
			ActorType:     attrs["actorType"],
			ActorLogin:    attrs["actorLogin"],
			ActorID:       attrs["actorID"],
			Event:         attrs["event"],
			WorkflowRunID: attrs["workflowRunID"],
		}
	}
}
