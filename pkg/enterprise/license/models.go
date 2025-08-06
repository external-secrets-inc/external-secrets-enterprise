// Copyright External Secrets Inc. 2025
//	All Rights Reserved

package license

import (
	"time"

	"github.com/golang-jwt/jwt"
)

type Feature struct {
}

// License represents the parsed license data
type License struct {
	Signature     string                `yaml:"signature"`
	Data          LicenseData           `yaml:"data"`
	Subscriptions []LicenseSubscription `yaml:"subscriptions"`
}

// LicenseData contains metadata about the license
type LicenseData struct {
	LicenseID      string    `yaml:"licenseId"`
	CustomerName   string    `yaml:"customerName"`
	IssueDate      time.Time `yaml:"issueDate"`
	ExpirationDate time.Time `yaml:"expirationDate"`
	Version        string    `yaml:"version"`
}

// LicenseSubscription represents a subscription in the license
type LicenseSubscription struct {
	Name       string `yaml:"name"`
	ExpiryDate string `yaml:"expiryDate"` // ISO8601 format
	MaxLimit   int    `yaml:"maxLimit"`
}

// LicenseClaims represents the JWT claims in the license
type LicenseClaims struct {
	LicenseID      string                `json:"licenseId"`
	CustomerName   string                `json:"customerName"`
	IssueDate      int64                 `json:"issueDate"`
	ExpirationDate int64                 `json:"expirationDate"`
	Version        string                `json:"version"`
	Subscriptions  []LicenseSubscription `json:"subscriptions"`
	jwt.StandardClaims
}
