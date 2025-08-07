// Copyright External Secrets Inc. 2025
//	All Rights Reserved

package license

import (
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"gopkg.in/yaml.v3"
)

// GenerateLicense creates a license file for testing purposes.
// Note: This function should only be used for development/testing.
func GenerateLicense(privateKey *rsa.PrivateKey, licenseData LicenseData, subscriptions []LicenseSubscription) ([]byte, error) {
	// Create JWT claims
	claims := LicenseClaims{
		LicenseID:      licenseData.LicenseID,
		CustomerName:   licenseData.CustomerName,
		IssueDate:      licenseData.IssueDate.Unix(),
		ExpirationDate: licenseData.ExpirationDate.Unix(),
		Version:        licenseData.Version,
		Subscriptions:  subscriptions,
	}

	// Create a new token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Sign the token with the private key
	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign license: %w", err)
	}

	// Create the license structure
	license := License{
		Signature:     signedToken,
		Data:          licenseData,
		Subscriptions: subscriptions,
	}

	// Marshal to YAML
	licenseYAML, err := yaml.Marshal(license)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal license to YAML: %w", err)
	}

	return licenseYAML, nil
}

// CreateSampleLicense creates a sample license for testing.
func CreateSampleLicense() License {
	return License{
		Signature: "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...", // This would be a real JWT in practice
		Data: LicenseData{
			LicenseID:      "lic-123456789",
			CustomerName:   "Acme Corporation",
			IssueDate:      time.Now(),
			ExpirationDate: time.Now().AddDate(1, 0, 0), // 1 year from now
			Version:        "1.0.0",
		},
		Subscriptions: []LicenseSubscription{},
	}
}
