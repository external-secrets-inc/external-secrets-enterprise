// Copyright External Secrets Inc. 2025
// All Rights Reserved

package job

import (
	"crypto/sha512"
	"encoding/hex"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/targets/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type MemorySet struct {
	mu          sync.RWMutex
	entries     map[tgtv1alpha1.SecretInStoreRef]string
	regexMap    map[string][]string
	valueToKeys map[string][]tgtv1alpha1.SecretInStoreRef
}

func NewMemorySet() *MemorySet {
	return &MemorySet{
		entries:     make(map[tgtv1alpha1.SecretInStoreRef]string),
		valueToKeys: make(map[string][]tgtv1alpha1.SecretInStoreRef),
		mu:          sync.RWMutex{},
	}
}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_-+="

var r *rand.Rand

func init() {
	source := rand.NewSource(time.Now().UnixNano())
	r = rand.New(source)
}

func generateRegexes(val []byte) []string {
	const (
		goodRegexes  = 10
		badRegexes   = 5
		charsPerRune = 7
	)

	regexes := make([]string, 0, goodRegexes+badRegexes)
	var sb strings.Builder

	// Generate regexes that are designed to match the input value
	for i := 0; i < goodRegexes; i++ {
		sb.Reset()
		for _, char := range val {
			sb.WriteString("[")
			sb.WriteByte(char)
			for j := 0; j < charsPerRune-1; j++ {
				sb.WriteByte(alphabet[r.Intn(len(alphabet))])
			}
			sb.WriteString("]")
		}
		regexes = append(regexes, sb.String())
	}

	// Generate regexes that are designed to not match the input value
	for i := 0; i < badRegexes; i++ {
		sb.Reset()
		for _, char := range val {
			sb.WriteString("[")
			for j := 0; j < charsPerRune; j++ {
				randomChar := alphabet[r.Intn(len(alphabet))]
				for randomChar == char {
					randomChar = alphabet[r.Intn(len(alphabet))]
				}
				sb.WriteByte(randomChar)
			}
			sb.WriteString("]")
		}
		regexes = append(regexes, sb.String())
	}

	r.Shuffle(len(regexes), func(i, j int) {
		regexes[i], regexes[j] = regexes[j], regexes[i]
	})

	return regexes
}

func (ms *MemorySet) Regexes() map[string][]string {
	return ms.regexMap
}

func (ms *MemorySet) AddByRegex(hash string, location tgtv1alpha1.SecretInStoreRef) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.valueToKeys[hash] = append(ms.valueToKeys[hash], location)
}

func (ms *MemorySet) Add(secret tgtv1alpha1.SecretInStoreRef, value []byte) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	h := hash(value)
	regs := generateRegexes(value)
	ms.entries[secret] = h
	ms.valueToKeys[h] = append(ms.valueToKeys[h], secret)
	ms.regexMap[h] = regs
}

func hash(value []byte) string {
	hash := sha512.Sum512(value)
	return hex.EncodeToString(hash[:])
}

// GetDuplicates now just scans the valueToKeys map to find values with more than one Entry.

func (ms *MemorySet) GetDuplicates() []v1alpha1.Finding {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var findings []v1alpha1.Finding
	for hash, keys := range ms.valueToKeys {
		if len(keys) > 1 {
			finding := v1alpha1.Finding{
				ObjectMeta: metav1.ObjectMeta{
					Name: hash[:16],
				},
				Spec: v1alpha1.FindingSpec{
					Hash: hash,
				},
			}
			for _, key := range keys {
				finding.Status.Locations = append(finding.Status.Locations, key)
			}
			findings = append(findings, finding)
		}
	}
	return findings
}
