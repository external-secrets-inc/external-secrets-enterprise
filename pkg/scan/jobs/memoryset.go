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

const (
	THRESHOLD    = 9
	GOOD_REGEXES = 10
	BAD_REGEXES  = 5
	charsPerRune = 7
)

type MemorySet struct {
	mu          sync.RWMutex
	entries     map[tgtv1alpha1.SecretInStoreRef]string
	regexMap    map[string][]string
	valueToKeys map[string][]tgtv1alpha1.SecretInStoreRef
	threshold   int
}

func NewMemorySet() *MemorySet {
	return &MemorySet{
		entries:     make(map[tgtv1alpha1.SecretInStoreRef]string),
		valueToKeys: make(map[string][]tgtv1alpha1.SecretInStoreRef),
		mu:          sync.RWMutex{},
		regexMap:    make(map[string][]string),
		// Todo flexibilize this
		threshold: THRESHOLD,
	}
}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*_+='\"{}"

var r *rand.Rand

func init() {
	source := rand.NewSource(time.Now().UnixNano())
	r = rand.New(source)
}

func generateRegexes(val []byte) []string {

	regexes := make([]string, 0, GOOD_REGEXES+BAD_REGEXES)
	var sb strings.Builder

	// Generate regexes that are designed to match the input value
	for i := 0; i < GOOD_REGEXES; i++ {
		sb.Reset()
		for _, char := range val {
			sb.WriteString("[")
			charSet := make([]byte, charsPerRune)
			charSet[0] = char
			for j := 1; j < charsPerRune; j++ {
				charSet[j] = alphabet[r.Intn(len(alphabet))]
			}

			r.Shuffle(len(charSet), func(i, j int) {
				charSet[i], charSet[j] = charSet[j], charSet[i]
			})

			sb.Write(charSet)
			sb.WriteString("]")
		}
		regexes = append(regexes, sb.String())
	}

	// Generate regexes that are designed to not match the input value
	for i := 0; i < BAD_REGEXES; i++ {
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

func (ms *MemorySet) GetThreshold() int {
	return ms.threshold
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
	// TODO: remove if I havent. This is troubleshooting
	// return string(value)
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
