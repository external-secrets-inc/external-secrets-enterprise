// Copyright External Secrets Inc. 2025
// All Rights Reserved

package job

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"math/big"
	"strings"
	"sync"

	"github.com/external-secrets/external-secrets/apis/enterprise/scan/v1alpha1"
	tgtv1alpha1 "github.com/external-secrets/external-secrets/apis/enterprise/targets/v1alpha1"
)

const (
	THRESHOLD    = 9
	GOOD_REGEXES = 10
	BAD_REGEXES  = 5
	charsPerRune = 7
)

type LocationMemorySet struct {
	mu          sync.RWMutex
	entries     map[tgtv1alpha1.SecretInStoreRef]string
	regexMap    map[string][]string
	valueToKeys map[string][]tgtv1alpha1.SecretInStoreRef
	threshold   int
}

func NewLocationMemorySet() *LocationMemorySet {
	return &LocationMemorySet{
		entries:     make(map[tgtv1alpha1.SecretInStoreRef]string),
		valueToKeys: make(map[string][]tgtv1alpha1.SecretInStoreRef),
		mu:          sync.RWMutex{},
		regexMap:    make(map[string][]string),
		// Todo flexibilize this
		threshold: THRESHOLD,
	}
}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

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
				n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
				charSet[j] = alphabet[n.Int64()]
			}

			// Fisher-Yates shuffle
			for k := len(charSet) - 1; k > 0; k-- {
				n, _ := rand.Int(rand.Reader, big.NewInt(int64(k+1)))
				j := n.Int64()
				charSet[k], charSet[j] = charSet[j], charSet[k]
			}

			// If the character is not in the alphabet, it's a special character.
			// We need to move it to the front of the charSet to avoid regex errors.
			if !strings.ContainsRune(alphabet, rune(char)) {
				for i, c := range charSet {
					if c == char {
						charSet[i], charSet[0] = charSet[0], charSet[i]
						break
					}
				}
			}

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
				var randomChar byte
				for {
					n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
					randomChar = alphabet[n.Int64()]
					if randomChar != char {
						break
					}
				}
				sb.WriteByte(randomChar)
			}
			sb.WriteString("]")
		}
		regexes = append(regexes, sb.String())
	}

	// Fisher-Yates shuffle for the regexes slice
	for i := len(regexes) - 1; i > 0; i-- {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		j := n.Int64()
		regexes[i], regexes[j] = regexes[j], regexes[i]
	}

	return regexes
}

func (ms *LocationMemorySet) Regexes() map[string][]string {
	return ms.regexMap
}

func (ms *LocationMemorySet) GetThreshold() int {
	return ms.threshold
}

func (ms *LocationMemorySet) AddByRegex(hash string, location tgtv1alpha1.SecretInStoreRef) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.valueToKeys[hash] = append(ms.valueToKeys[hash], location)
}

func (ms *LocationMemorySet) Add(secret tgtv1alpha1.SecretInStoreRef, value []byte) {
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

func (ms *LocationMemorySet) GetDuplicates() []v1alpha1.Finding {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	findings := make([]v1alpha1.Finding, 0, len(ms.valueToKeys))
	for hash, keys := range ms.valueToKeys {
		if len(keys) < 2 {
			continue
		}

		finding := v1alpha1.Finding{
			Spec: v1alpha1.FindingSpec{
				Hash: hash,
			},
		}
		for _, key := range keys {
			finding.Status.Locations = append(finding.Status.Locations, key)
		}
		SortLocations(finding.Status.Locations)
		finding.Spec.Label = Sanitize(finding.Status.Locations[0])
		findings = append(findings, finding)
	}
	return findings
}
