// Copyright External Secrets Inc. 2025
// All rights reserved

package targets

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	utils "github.com/external-secrets/external-secrets/pkg/enterprise/scan/jobs"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const maxHistoryPerLocation = 20

func UpdateTargetPushIndex(
	ctx context.Context,
	objKind string,
	kubeClient client.Client,
	name string,
	namespace string,
	key string,
	property string,
	hash string,
) error {
	if kubeClient == nil {
		return fmt.Errorf("kube client is not configured on ScanTarget")
	}

	locationKey := key
	if strings.TrimSpace(property) != "" {
		locationKey = fmt.Sprintf("%s.%s", key, property)
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		gvk := schema.GroupVersionKind{Group: tgtv1alpha1.Group, Version: tgtv1alpha1.Version, Kind: objKind}
		obj, err := kubeClient.Scheme().New(gvk)
		if err != nil {
			return fmt.Errorf("failed to create object %v: %w", gvk, err)
		}
		genericTarget, ok := obj.(tgtv1alpha1.GenericTarget)
		if !ok {
			return fmt.Errorf("invalid object: %T", obj)
		}
		err = kubeClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, genericTarget)
		if err != nil {
			return fmt.Errorf("failed to get object %s/%s: %w", namespace, name, err)
		}

		status := genericTarget.GetTargetStatus()
		if status.PushIndex == nil {
			status.PushIndex = make(map[string][]scanv1alpha1.SecretUpdateRecord, 1)
		}

		hist := status.PushIndex[locationKey]

		hist = append(hist, scanv1alpha1.SecretUpdateRecord{
			Timestamp:  metav1.NewTime(metav1.Now().UTC()),
			SecretHash: hash,
		})

		if len(hist) > maxHistoryPerLocation {
			hist = hist[len(hist)-maxHistoryPerLocation:]
		}
		status.PushIndex[locationKey] = hist
		genericTarget.SetTargetStatus(status)

		return kubeClient.Status().Update(ctx, genericTarget)
	})
}

func Hash(value []byte) string {
	hash := sha512.Sum512(value)
	return hex.EncodeToString(hash[:])
}

func UpdateConsumersFromFindings(
	ctx context.Context,
	k8s client.Client,
	targetNamespace string,
	consumerFindings []scanv1alpha1.ConsumerFinding,
) error {
	type key struct{ id, obsKey string }
	latestConsumerFinding := make(map[key]scanv1alpha1.ConsumerFinding)

	for _, consumerFinding := range consumerFindings {
		obsKey := consumerFinding.Location.RemoteRef.Key
		if property := consumerFinding.Location.RemoteRef.Property; property != "" {
			obsKey = obsKey + "." + property
		}
		k := key{id: consumerFinding.ID, obsKey: obsKey}

		if prev, ok := latestConsumerFinding[k]; !ok ||
			consumerFinding.ObservedIndex.Timestamp.Time.After(prev.ObservedIndex.Timestamp.Time) {
			latestConsumerFinding[k] = consumerFinding
		}
	}

	consumerFindingGroup := make(map[string][]scanv1alpha1.ConsumerFinding)
	for _, consumerFinding := range latestConsumerFinding {
		consumerFindingGroup[consumerFinding.ID] = append(consumerFindingGroup[consumerFinding.ID], consumerFinding)
	}

	for id := range consumerFindingGroup {
		if err := applyObservedIndex(ctx, k8s, targetNamespace, id, consumerFindingGroup[id]); err != nil {
			return err
		}
	}
	return nil
}

func applyObservedIndex(
	ctx context.Context,
	k8s client.Client,
	namespace, consumerName string,
	consumerFindings []scanv1alpha1.ConsumerFinding,
) error {
	var consumer scanv1alpha1.Consumer
	nn := types.NamespacedName{Namespace: namespace, Name: consumerName}
	if err := k8s.Get(ctx, nn, &consumer); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	orig := consumer.DeepCopy()

	if consumer.Status.ObservedIndex == nil {
		consumer.Status.ObservedIndex = make(map[string]scanv1alpha1.SecretUpdateRecord)
	}

	for _, consumerFinding := range consumerFindings {
		obsKey := consumerFinding.Location.RemoteRef.Key
		if property := consumerFinding.Location.RemoteRef.Property; property != "" {
			obsKey = obsKey + "." + property
		}
		cur, has := consumer.Status.ObservedIndex[obsKey]
		if !has || consumerFinding.ObservedIndex.Timestamp.Time.After(cur.Timestamp.Time) {
			consumer.Status.ObservedIndex[obsKey] = consumerFinding.ObservedIndex
		}

		if !hasLocation(consumer.Status.Locations, consumerFinding.Location) {
			consumer.Status.Locations = append(consumer.Status.Locations, consumerFinding.Location)
		}
	}

	if equalObservedIndexMaps(orig.Status.ObservedIndex, consumer.Status.ObservedIndex) &&
		len(consumer.Status.Locations) == len(orig.Status.Locations) {
		return nil
	}

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := k8s.Get(ctx, nn, &consumer); err != nil {
			return err
		}
		if consumer.Status.ObservedIndex == nil {
			consumer.Status.ObservedIndex = make(map[string]scanv1alpha1.SecretUpdateRecord)
		}
		for _, consumerFinding := range consumerFindings {
			obsKey := consumerFinding.Location.RemoteRef.Key
			if property := consumerFinding.Location.RemoteRef.Property; property != "" {
				obsKey = obsKey + "." + property
			}
			cur, has := consumer.Status.ObservedIndex[obsKey]
			if !has || consumerFinding.ObservedIndex.Timestamp.Time.After(cur.Timestamp.Time) {
				consumer.Status.ObservedIndex[obsKey] = consumerFinding.ObservedIndex
			}
			if !hasLocation(consumer.Status.Locations, consumerFinding.Location) {
				consumer.Status.Locations = append(consumer.Status.Locations, consumerFinding.Location)
			}
		}
		return k8s.Status().Update(ctx, &consumer)
	})
}

func hasLocation(list []scanv1alpha1.SecretInStoreRef, want scanv1alpha1.SecretInStoreRef) bool {
	for _, secretInStoreRef := range list {
		if utils.EqualLocations(secretInStoreRef, want) {
			return true
		}
	}
	return false
}

func equalObservedIndexMaps(a, b map[string]scanv1alpha1.SecretUpdateRecord) bool {
	if len(a) != len(b) {
		return false
	}
	for key, secretUpdateRecordA := range a {
		secretUpdateRecordB, ok := b[key]
		if !ok {
			return false
		}
		// compare fields you store (hash, ts, etc.)
		if !secretUpdateRecordA.Timestamp.Time.Equal(secretUpdateRecordB.Timestamp.Time) || secretUpdateRecordA.SecretHash != secretUpdateRecordB.SecretHash {
			return false
		}
	}
	return true
}
