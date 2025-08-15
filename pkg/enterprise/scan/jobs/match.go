package job

import (
	"fmt"

	"github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Candidate struct {
	ID        string
	Name      string
	Inter     int
	Union     int
	Jaccard   float64
	CurrCount int
}

type Params struct {
	MinJaccard      float64 // e.g. 0.6
	MinIntersection int     // e.g. 2
}

func Jaccard(a, b map[string]struct{}) (inter, uni int, j float64) {
	// compute intersection and union
	if len(a) < len(b) {
		a, b = b, a
	}
	inter = 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	uni = len(a) + len(b) - inter
	if uni == 0 {
		return 0, 0, 1.0
	}
	return inter, uni, float64(inter) / float64(uni)
}

func LocationsToSet(locations []tgtv1alpha1.SecretInStoreRef) map[string]struct{} {
	m := make(map[string]struct{}, len(locations))
	for _, k := range locations {
		m[Sanitize(k)] = struct{}{}
	}
	return m
}

func AssignIDs(currentFindings []v1alpha1.Finding, newFindings []v1alpha1.Finding, p Params) []v1alpha1.Finding {
	// Pre-sort not necessary; we’ll compute best per new
	for i := range newFindings {
		newLocationsSet := LocationsToSet(newFindings[i].Status.Locations)
		best := Candidate{}
		for _, currentFinding := range currentFindings {
			currentLocationsSet := LocationsToSet(currentFinding.Status.Locations)
			inter, uni, j := Jaccard(currentLocationsSet, newLocationsSet)
			ok := j >= p.MinJaccard || (inter >= p.MinIntersection && inter*2 >= min(len(newLocationsSet), len(currentLocationsSet)))
			if !ok {
				continue
			}
			cand := Candidate{
				ID:        currentFinding.Spec.ID,
				Name:      currentFinding.Name,
				Inter:     inter,
				Union:     uni,
				Jaccard:   j,
				CurrCount: len(currentLocationsSet),
			}
			if better(cand, best) {
				best = cand
			}
		}
		if best.ID != "" {
			newFindings[i].Spec.ID = best.ID
			newFindings[i].ObjectMeta = metav1.ObjectMeta{
				Name: best.Name,
			}
		} else {
			newUUID := uuid.NewString()
			newFindings[i].Spec.ID = newUUID
			newFindings[i].ObjectMeta = metav1.ObjectMeta{
				Name: fmt.Sprintf("finding-%s", newUUID),
			}
		}
	}
	return newFindings
}

func better(a, b Candidate) bool {
	if b.ID == "" {
		return true
	}
	if a.Jaccard != b.Jaccard {
		return a.Jaccard > b.Jaccard
	}
	if a.Inter != b.Inter {
		return a.Inter > b.Inter
	}
	// deterministic tie-breaker
	return a.ID < b.ID
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
