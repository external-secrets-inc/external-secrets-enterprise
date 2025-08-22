// Copyright External Secrets Inc. 2025
// All Rights Reserved

package job

import (
	"slices"
	"strings"
	"sync"

	"github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConsumerKey struct {
	TargetNS string
	Target   string
	Type     string
	ID       string
}

// Accumulator for a single consumer.
type consumerAccum struct {
	spec   v1alpha1.ConsumerSpec
	status v1alpha1.ConsumerStatus
}

// ConsumerMemorySet merges many Provider findings into per-consumer accumulators.
type ConsumerMemorySet struct {
	mu     sync.RWMutex
	accums map[ConsumerKey]*consumerAccum
}

func NewConsumerMemorySet() *ConsumerMemorySet {
	return &ConsumerMemorySet{
		accums: make(map[ConsumerKey]*consumerAccum),
	}
}

func (cs *ConsumerMemorySet) Add(target v1alpha1.TargetReference, f tgtv1alpha1.ConsumerFinding) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	key := ConsumerKey{
		TargetNS: target.Namespace,
		Target:   target.Name,
		Type:     f.Kind,
		ID:       f.ID,
	}
	acc, ok := cs.accums[key]
	if !ok {
		acc = &consumerAccum{
			spec: v1alpha1.ConsumerSpec{
				Target:      target,
				Type:        f.Kind,
				ID:          f.ID,
				DisplayName: f.DisplayName,
			},
			status: v1alpha1.ConsumerStatus{},
		}
		FillAttributes(acc, f.Kind, f.Attributes)
		cs.accums[key] = acc
	}

	already := false
	for _, loc := range acc.status.Locations {
		if EqualLocations(loc, f.Location) {
			already = true
			break
		}
	}
	if !already {
		acc.status.Locations = append(acc.status.Locations, f.Location)
	}
}

func (cs *ConsumerMemorySet) List() []v1alpha1.Consumer {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	out := make([]v1alpha1.Consumer, 0, len(cs.accums))
	for _, acc := range cs.accums {
		SortLocations(acc.status.Locations)
		slices.SortFunc(acc.status.Pods, func(a, b v1alpha1.K8sPodItem) int {
			if a.UID == b.UID {
				return strings.Compare(a.Name, b.Name)
			}
			return strings.Compare(a.UID, b.UID)
		})
		out = append(out, v1alpha1.Consumer{
			ObjectMeta: metav1.ObjectMeta{
				Name: acc.spec.ID,
			},
			Spec:   acc.spec,
			Status: acc.status,
		})
	}
	return out
}
