// Copyright External Secrets Inc. 2025
//
//	All Rights Reserved
package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/external-secrets/external-secrets/pkg/enterprise/license"
)

// Test helpers and fixtures

func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate test RSA key")
	return privateKey
}

func createTestLicenseData() license.LicenseData {
	now := time.Now()
	return license.LicenseData{
		LicenseID:      "test-license-123",
		CustomerName:   "Test Customer",
		IssueDate:      now,
		ExpirationDate: now.AddDate(0, 1, 0), // 1 month from now
		Version:        "test-1.0",
	}
}

func createTestSubscriptions() []license.LicenseSubscription {
	date := time.Now()
	return []license.LicenseSubscription{
		{
			Name:       "test.feature.unlimited",
			ExpiryDate: date.String(),
			MaxLimit:   0,
		},
		{
			Name:       "test.feature.limited",
			ExpiryDate: date.String(),
			MaxLimit:   5,
		},
	}
}

// Key Generation Tests

func TestGenerateRSAKeys_ValidKeyPair(t *testing.T) {
	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate RSA key pair")

	// Verify key is valid
	assert.NotNil(t, privateKey, "Private key should not be nil")
	assert.NotNil(t, &privateKey.PublicKey, "Public key should not be nil")
	assert.Equal(t, 2048, privateKey.N.BitLen(), "Key should be 2048 bits")

	// Verify private key is valid
	err = privateKey.Validate()
	assert.NoError(t, err, "Private key should be valid")
}

func TestGenerateRSAKeys_KeySize_2048(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate RSA key")

	// Verify the key size is exactly 2048 bits
	assert.Equal(t, 2048, privateKey.N.BitLen(), "Key size should be 2048 bits")

	// Verify the key can be used for signing
	hash := []byte("test hash for signing verification")
	_, err = rsa.SignPKCS1v15(rand.Reader, privateKey, 0, hash)
	assert.NoError(t, err, "Key should be usable for signing")
}

func TestVerifyGeneratedKeys_SignAndVerify(t *testing.T) {
	// Generate key pair
	privateKey := generateTestRSAKey(t)
	publicKey := &privateKey.PublicKey

	// Create test data
	testData := "test message for signing"

	// Create JWT token with test claims
	claims := jwt.MapClaims{
		"test": testData,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	// Sign with private key
	tokenString, err := token.SignedString(privateKey)
	require.NoError(t, err, "Failed to sign token")
	assert.NotEmpty(t, tokenString, "Signed token should not be empty")

	// Verify with public key
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	require.NoError(t, err, "Failed to verify token signature")
	assert.True(t, parsedToken.Valid, "Token should be valid")
}

// License Generation Tests

func TestGenerateLicense_ValidInputs(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	licenseData := createTestLicenseData()
	subscriptions := createTestSubscriptions()

	// Generate license
	generatedLicense, err := license.GenerateLicense(privateKey, licenseData, subscriptions)
	require.NoError(t, err, "Failed to generate license")
	assert.NotEmpty(t, generatedLicense, "Generated license should not be empty")

	// Parse the generated YAML
	var parsedLicense license.License
	err = yaml.Unmarshal(generatedLicense, &parsedLicense)
	require.NoError(t, err, "Failed to parse generated license YAML")

	// Verify license structure
	assert.NotEmpty(t, parsedLicense.Signature, "License signature should not be empty")
	assert.Equal(t, licenseData.LicenseID, parsedLicense.Data.LicenseID, "License ID should match")
	assert.Equal(t, licenseData.CustomerName, parsedLicense.Data.CustomerName, "Customer name should match")
	assert.Equal(t, len(subscriptions), len(parsedLicense.Subscriptions), "Subscription count should match")
}

func TestGenerateLicense_AllSubscriptionTypes(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	licenseData := createTestLicenseData()

	// Create subscriptions with different types
	subscriptions := []license.LicenseSubscription{
		{Name: "unlimited.feature", MaxLimit: 0}, // Unlimited
		{Name: "limited.feature", MaxLimit: 10},  // Limited
		{Name: "single.feature", MaxLimit: 1},    // Single use
	}

	generatedLicense, err := license.GenerateLicense(privateKey, licenseData, subscriptions)
	require.NoError(t, err, "Failed to generate license with all subscription types")

	// Parse and verify all subscription types are preserved
	var parsedLicense license.License
	err = yaml.Unmarshal(generatedLicense, &parsedLicense)
	require.NoError(t, err, "Failed to parse license")

	assert.Len(t, parsedLicense.Subscriptions, 3, "Should have 3 subscriptions")

	// Verify subscription limits are preserved
	for i, sub := range parsedLicense.Subscriptions {
		assert.Equal(t, subscriptions[i].MaxLimit, sub.MaxLimit,
			"Subscription %s limit should be preserved", sub.Name)
	}
}

func TestGenerateLicense_TrialLicenseGeneration(t *testing.T) {
	privateKey := generateTestRSAKey(t)

	// Use the same trial license data structure as main.go
	date := time.Now()
	trialLicData := license.LicenseData{
		LicenseID:      "trial-license",
		CustomerName:   "trial",
		IssueDate:      date,
		ExpirationDate: date.AddDate(0, 1, 0), // 1 month from now
		Version:        "trial",
	}

	// Use some of the actual trial subscriptions
	trialSubs := []license.LicenseSubscription{
		{Name: "workflow.core", ExpiryDate: date.String(), MaxLimit: 0},
		{Name: "generator.aws_iam", ExpiryDate: date.String(), MaxLimit: 0},
	}

	generatedLicense, err := license.GenerateLicense(privateKey, trialLicData, trialSubs)
	require.NoError(t, err, "Failed to generate trial license")

	// Parse and verify trial license structure
	var parsedLicense license.License
	err = yaml.Unmarshal(generatedLicense, &parsedLicense)
	require.NoError(t, err, "Failed to parse trial license")

	assert.Equal(t, "trial-license", parsedLicense.Data.LicenseID, "Trial license ID should match")
	assert.Equal(t, "trial", parsedLicense.Data.CustomerName, "Trial customer name should match")
	assert.Equal(t, "trial", parsedLicense.Data.Version, "Trial version should match")
}

func TestGenerateLicense_ExpiryDateCalculation(t *testing.T) {
	privateKey := generateTestRSAKey(t)

	// Create license with specific expiry date
	issueDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedExpiry := issueDate.AddDate(1, 0, 0) // 1 year later

	licenseData := license.LicenseData{
		LicenseID:      "expiry-test-license",
		CustomerName:   "Expiry Test Customer",
		IssueDate:      issueDate,
		ExpirationDate: expectedExpiry,
		Version:        "1.0",
	}

	generatedLicense, err := license.GenerateLicense(privateKey, licenseData, []license.LicenseSubscription{})
	require.NoError(t, err, "Failed to generate license with expiry date")

	// Parse and verify expiry date
	var parsedLicense license.License
	err = yaml.Unmarshal(generatedLicense, &parsedLicense)
	require.NoError(t, err, "Failed to parse license")

	assert.Equal(t, expectedExpiry.Unix(), parsedLicense.Data.ExpirationDate.Unix(),
		"Expiry date should match expected date")
	assert.True(t, parsedLicense.Data.ExpirationDate.After(issueDate),
		"Expiry date should be after issue date")
}

// JWT Creation Tests

func TestCreateJWTClaims_AllFields(t *testing.T) {
	licenseData := createTestLicenseData()
	subscriptions := createTestSubscriptions()

	// Create claims manually (similar to GenerateLicense function)
	claims := license.LicenseClaims{
		LicenseID:      licenseData.LicenseID,
		CustomerName:   licenseData.CustomerName,
		IssueDate:      licenseData.IssueDate.Unix(),
		ExpirationDate: licenseData.ExpirationDate.Unix(),
		Version:        licenseData.Version,
		Subscriptions:  subscriptions,
	}

	// Verify all fields are populated
	assert.Equal(t, licenseData.LicenseID, claims.LicenseID, "License ID should match")
	assert.Equal(t, licenseData.CustomerName, claims.CustomerName, "Customer name should match")
	assert.Equal(t, licenseData.IssueDate.Unix(), claims.IssueDate, "Issue date should match")
	assert.Equal(t, licenseData.ExpirationDate.Unix(), claims.ExpirationDate, "Expiration date should match")
	assert.Equal(t, licenseData.Version, claims.Version, "Version should match")
	assert.Equal(t, len(subscriptions), len(claims.Subscriptions), "Subscriptions count should match")

	// Verify Unix timestamp conversion
	assert.Greater(t, claims.IssueDate, int64(0), "Issue date Unix timestamp should be positive")
	assert.Greater(t, claims.ExpirationDate, claims.IssueDate, "Expiration should be after issue date")
}

func TestSignJWT_ValidSignature(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	claims := license.LicenseClaims{
		LicenseID:      "test-jwt-signature",
		CustomerName:   "JWT Test Customer",
		IssueDate:      time.Now().Unix(),
		ExpirationDate: time.Now().Add(time.Hour).Unix(),
		Version:        "jwt-test",
	}

	// Create and sign token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	require.NoError(t, err, "Failed to sign JWT")
	assert.NotEmpty(t, signedToken, "Signed token should not be empty")

	// Verify token structure (should have 3 parts separated by dots)
	parts := strings.Split(signedToken, ".")
	assert.Len(t, parts, 3, "JWT should have 3 parts (header.payload.signature)")

	// Each part should be non-empty
	for i, part := range parts {
		assert.NotEmpty(t, part, "JWT part %d should not be empty", i)
	}
}

func TestSignJWT_VerifyWithGeneratedKeys(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	publicKey := &privateKey.PublicKey

	claims := license.LicenseClaims{
		LicenseID:      "verification-test",
		CustomerName:   "Verification Customer",
		IssueDate:      time.Now().Unix(),
		ExpirationDate: time.Now().Add(time.Hour).Unix(),
		Version:        "verify-1.0",
		Subscriptions:  createTestSubscriptions(),
	}

	// Sign token
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signedToken, err := token.SignedString(privateKey)
	require.NoError(t, err, "Failed to sign token")

	// Verify and parse token
	parsedToken, err := jwt.ParseWithClaims(signedToken, &license.LicenseClaims{}, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})
	require.NoError(t, err, "Failed to verify token")
	assert.True(t, parsedToken.Valid, "Token should be valid")

	// Verify claims
	parsedClaims, ok := parsedToken.Claims.(*license.LicenseClaims)
	require.True(t, ok, "Claims should be of correct type")
	assert.Equal(t, claims.LicenseID, parsedClaims.LicenseID, "License ID should match")
	assert.Equal(t, claims.CustomerName, parsedClaims.CustomerName, "Customer name should match")
	assert.Equal(t, len(claims.Subscriptions), len(parsedClaims.Subscriptions), "Subscriptions should match")
}

// Template Rendering Tests

func TestRenderTemplate_ValidOutput(t *testing.T) {
	// Test data for template
	tmplData := struct {
		PublicKey   string
		LicenseData string
	}{
		PublicKey:   "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...\n-----END PUBLIC KEY-----\n",
		LicenseData: "signature: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...\ndata:\n  licenseId: test-license\n",
	}

	// Parse template
	tmpl, err := template.New("license").Parse(generatedFile)
	require.NoError(t, err, "Failed to parse template")

	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, tmplData)
	require.NoError(t, err, "Failed to execute template")

	output := buf.String()
	assert.NotEmpty(t, output, "Template output should not be empty")

	// Verify output contains expected elements
	assert.Contains(t, output, "package license", "Output should contain package declaration")
	assert.Contains(t, output, "const trialLicensePubKey", "Output should contain public key constant")
	assert.Contains(t, output, "const trialLicenseData", "Output should contain license data constant")
	assert.Contains(t, output, "Code generated by go generate", "Output should contain generation comment")
}

func TestRenderTemplate_AllVariablesReplaced(t *testing.T) {
	testPublicKey := "TEST_PUBLIC_KEY_CONTENT"
	testLicenseData := "TEST_LICENSE_DATA_CONTENT"

	tmplData := struct {
		PublicKey   string
		LicenseData string
	}{
		PublicKey:   testPublicKey,
		LicenseData: testLicenseData,
	}

	// Parse and execute template
	tmpl, err := template.New("license").Parse(generatedFile)
	require.NoError(t, err, "Failed to parse template")

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, tmplData)
	require.NoError(t, err, "Failed to execute template")

	output := buf.String()

	// Verify all variables are replaced
	assert.Contains(t, output, testPublicKey, "Public key should be replaced in template")
	assert.Contains(t, output, testLicenseData, "License data should be replaced in template")
	assert.NotContains(t, output, "{{.PublicKey}}", "Template variables should be replaced")
	assert.NotContains(t, output, "{{.LicenseData}}", "Template variables should be replaced")
}

// Integration with Trial Arrays Tests

func TestTrialArrays_NotEmpty(t *testing.T) {
	// Verify all trial arrays are not empty
	assert.NotEmpty(t, workflowTrial, "workflowTrial array should not be empty")
	assert.NotEmpty(t, federationTrial, "federationTrial array should not be empty")
	assert.NotEmpty(t, scanTrial, "scanTrial array should not be empty")
	assert.NotEmpty(t, targetsTrial, "targetsTrial array should not be empty")
	assert.NotEmpty(t, generatorTrial, "generatorTrial array should not be empty")

	// Verify each array has at least one subscription
	assert.Greater(t, len(workflowTrial), 0, "workflowTrial should have subscriptions")
	assert.Greater(t, len(federationTrial), 0, "federationTrial should have subscriptions")
	assert.Greater(t, len(scanTrial), 0, "scanTrial should have subscriptions")
	assert.Greater(t, len(targetsTrial), 0, "targetsTrial should have subscriptions")
	assert.Greater(t, len(generatorTrial), 0, "generatorTrial should have subscriptions")
}

func TestTrialArrays_AllFeaturesCovered(t *testing.T) {
	// Collect all subscription names
	allSubscriptions := make(map[string]bool)

	// Add all subscriptions from trial arrays
	for _, sub := range workflowTrial {
		allSubscriptions[sub.Name] = true
	}
	for _, sub := range federationTrial {
		allSubscriptions[sub.Name] = true
	}
	for _, sub := range scanTrial {
		allSubscriptions[sub.Name] = true
	}
	for _, sub := range targetsTrial {
		allSubscriptions[sub.Name] = true
	}
	for _, sub := range generatorTrial {
		allSubscriptions[sub.Name] = true
	}

	// Verify we have a reasonable number of features covered
	assert.Greater(t, len(allSubscriptions), 10, "Should have more than 10 different features covered")

	// Verify each feature category is represented
	hasWorkflow := false
	hasFederation := false
	hasScan := false
	hasTargets := false
	hasGenerator := false

	for name := range allSubscriptions {
		if strings.HasPrefix(name, "workflow.") {
			hasWorkflow = true
		}
		if strings.HasPrefix(name, "federation.") {
			hasFederation = true
		}
		if strings.HasPrefix(name, "scan.") {
			hasScan = true
		}
		if strings.HasPrefix(name, "targets.") {
			hasTargets = true
		}
		if strings.HasPrefix(name, "generator.") {
			hasGenerator = true
		}
	}

	assert.True(t, hasWorkflow, "Should have workflow features")
	assert.True(t, hasFederation, "Should have federation features")
	assert.True(t, hasScan, "Should have scan features")
	assert.True(t, hasTargets, "Should have targets features")
	assert.True(t, hasGenerator, "Should have generator features")
}

func TestTrialArrays_ValidSubscriptionNames(t *testing.T) {
	allArrays := []struct {
		name          string
		subscriptions []license.LicenseSubscription
	}{
		{"workflowTrial", workflowTrial},
		{"federationTrial", federationTrial},
		{"scanTrial", scanTrial},
		{"targetsTrial", targetsTrial},
		{"generatorTrial", generatorTrial},
	}

	for _, array := range allArrays {
		for _, sub := range array.subscriptions {
			// Verify subscription name is not empty
			assert.NotEmpty(t, sub.Name, "Subscription name should not be empty in %s", array.name)

			// Verify subscription name follows expected pattern (category.feature)
			assert.Contains(t, sub.Name, ".", "Subscription name should contain category separator in %s", array.name)

			// Verify expiry date is set
			assert.NotEmpty(t, sub.ExpiryDate, "Expiry date should not be empty in %s for %s", array.name, sub.Name)

			// Verify max limit is reasonable (0 for unlimited, >0 for limited)
			assert.GreaterOrEqual(t, sub.MaxLimit, 0, "Max limit should be non-negative in %s for %s", array.name, sub.Name)
		}
	}
}
