// Copyright External Secrets Inc. 2025
// All Rights Reserved

package job

import (
	"fmt"
	"slices"
	"strings"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
)

func Sanitize(ref scanv1alpha1.SecretInStoreRef) string {
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

func EqualLocations(a, b scanv1alpha1.SecretInStoreRef) bool {
	return a.Name == b.Name && a.Kind == b.Kind && a.APIVersion == b.APIVersion && a.RemoteRef.Key == b.RemoteRef.Key && a.RemoteRef.Property == b.RemoteRef.Property
}

func CompareLocations(a, b scanv1alpha1.SecretInStoreRef) int {
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

func SortLocations(loc []scanv1alpha1.SecretInStoreRef) {
	slices.SortFunc(loc, CompareLocations)
}

func EqualSecretUpdateRecord(a, b scanv1alpha1.SecretUpdateRecord) bool {
	return a.SecretHash == b.SecretHash && a.Timestamp.Equal(&b.Timestamp)
}
