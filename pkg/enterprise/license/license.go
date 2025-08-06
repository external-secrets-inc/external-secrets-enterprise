// Copyright External Secrets Inc. 2025
//
//	All Rights Reserved
package license

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/external-secrets/external-secrets/pkg/enterprise/license/feature"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/gommon/log"
	"gopkg.in/yaml.v3"
)

// LicenseService handles license validation and management
var licenseFilePath string
var license *License

func init() {
	keys["trial"] = trialLicensePubKey
	licenseFilePath = os.Getenv("ESI_LICENSE_FILE")
	if licenseFilePath == "" {
		licenseFilePath = "./license.yaml"
	}
	// Read the license file
	licenseData, err := os.ReadFile(licenseFilePath)
	if err != nil {
		// Default to Baseline Trial License Key
		log.Warn("Thanks for using External Secrets Inc. in its Trial License! contact https://externalsecrets.com for a full fledged license")
		licenseData = []byte(trialLicenseData)
	}

	// Parse the license YAML
	var lic License
	if err := yaml.Unmarshal(licenseData, &lic); err != nil {
		panic("failed to parse license file: " + err.Error())
	}

	// Validate the license signature
	if err := defaultValidateSignature(&lic); err != nil {
		panic("license validation failed: " + err.Error())
	}

	// Check if the license is expired
	if time.Now().After(lic.Data.ExpirationDate) {
		log.Warn("You are running with an expired license. Please renew your license!")
	}

	license = &lic
}

func Register(feature feature.Feature) error {
	for _, sub := range license.Subscriptions {
		if sub.Name == feature.Name() {
			// We must register this feature
			feature.SetMaxLimit(sub.MaxLimit)
			feature.SetExpiryDate(sub.ExpiryDate)
			err := feature.Enable()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// defaultValidateSignature verifies the JWT signature of the license
func defaultValidateSignature(license *License) error {
	// Parse the public key
	if _, ok := keys[license.Data.Version]; !ok {
		return fmt.Errorf("unknown license version: %s", license.Data.Version)
	}
	key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(keys[license.Data.Version]))
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Parse and validate the JWT token
	token, err := jwt.ParseWithClaims(license.Signature, &LicenseClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return key, nil
	})

	if err != nil {
		return fmt.Errorf("failed to validate license signature: %w", err)
	}

	// Extract the claims
	if claims, ok := token.Claims.(*LicenseClaims); ok && token.Valid {
		// Update the license data from the validated claims
		license.Data.LicenseID = claims.LicenseID
		license.Data.CustomerName = claims.CustomerName
		license.Data.IssueDate = time.Unix(claims.IssueDate, 0)
		license.Data.ExpirationDate = time.Unix(claims.ExpirationDate, 0)
		license.Data.Version = claims.Version
		license.Subscriptions = claims.Subscriptions

		return nil
	}

	return errors.New("invalid license claims")
}
