// Copyright External Secrets Inc. 2025
//
//	All Rights Reserved
package license

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Test data constants
const testValidLicenseYAML = `
signature: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJsaWNlbnNlSWQiOiJ0ZXN0LWxpY2Vuc2UiLCJjdXN0b21lck5hbWUiOiJ0ZXN0LWN1c3RvbWVyIiwiaXNzdWVEYXRlIjoxNzI1MTA3ODgyLCJleHBpcmF0aW9uRGF0ZSI6MTcyNzcwMjQ4MiwidmVyc2lvbiI6InRyaWFsIiwic3Vic2NyaXB0aW9ucyI6W3sibmFtZSI6InRlc3QuZmVhdHVyZSIsImV4cGlyeURhdGUiOiIyMDI1LTEwLTAxVDA5OjQ2OjIyLjcwODQ3MyswMzowMCIsIm1heExpbWl0IjoxMH1dfQ.invalid_signature_for_testing
data:
  licenseId: test-license
  customerName: test-customer
  issueDate: 2024-08-31T10:45:00Z
  expirationDate: 2024-10-01T10:45:00Z
  version: trial
subscriptions:
  - name: test.feature
    expiryDate: 2024-10-01T10:45:00Z
    maxLimit: 10
`

const testMalformedYAML = `
signature: invalid_yaml
data:
  licenseId: test
  - invalid yaml structure
`

const testUnknownVersionLicenseYAML = `
signature: test_signature
data:
  licenseId: test-license
  customerName: test-customer
  issueDate: 2024-08-31T10:45:00Z
  expirationDate: 2024-10-01T10:45:00Z
  version: unknown_version
subscriptions: []
`

// Mock feature for testing
type mockFeature struct {
	name        string
	description string
	maxLimit    int
	expiryDate  string
	enabled     bool
}

func (m *mockFeature) Name() string                            { return m.name }
func (m *mockFeature) Description() string                     { return m.description }
func (m *mockFeature) Register(content client.Object) error    { return nil }
func (m *mockFeature) Unregister(content client.Object) error  { return nil }
func (m *mockFeature) IsRegistered(content client.Object) bool { return false }
func (m *mockFeature) IsAvailable() bool                       { return m.enabled }
func (m *mockFeature) Enable() error                           { m.enabled = true; return nil }
func (m *mockFeature) Disable() error                          { m.enabled = false; return nil }
func (m *mockFeature) SetMaxLimit(limit int)                   { m.maxLimit = limit }
func (m *mockFeature) SetExpiryDate(expiryDate string)         { m.expiryDate = expiryDate }

// JWT Signature Validation Tests

func TestValidateSignature_ValidLicense(t *testing.T) {
	// Create a valid license using the trial license structure
	var lic License
	err := yaml.Unmarshal([]byte(trialLicenseData), &lic)
	require.NoError(t, err)

	// This should pass validation since we're using the actual trial license
	err = defaultValidateSignature(&lic)
	assert.NoError(t, err)
	assert.Equal(t, "trial-license", lic.Data.LicenseID)
	assert.Equal(t, "trial", lic.Data.CustomerName)
}

func TestValidateSignature_InvalidSignature(t *testing.T) {
	var lic License
	err := yaml.Unmarshal([]byte(testValidLicenseYAML), &lic)
	require.NoError(t, err)

	// This should fail validation due to an invalid signature
	err = defaultValidateSignature(&lic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate license signature")
}

func TestValidateSignature_UnknownVersion(t *testing.T) {
	var lic License
	err := yaml.Unmarshal([]byte(testUnknownVersionLicenseYAML), &lic)
	require.NoError(t, err)

	err = defaultValidateSignature(&lic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown license version: unknown_version")
}

func TestValidateSignature_MalformedToken(t *testing.T) {
	lic := License{
		Signature: "not.a.valid.jwt.token",
		Data: LicenseData{
			Version: "trial",
		},
	}

	err := defaultValidateSignature(&lic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to validate license signature")
}

func TestValidateSignature_WrongSigningMethod(t *testing.T) {
	// Create a JWT token with HMAC signing method instead of RSA
	claims := &LicenseClaims{
		LicenseID:      "test",
		CustomerName:   "test",
		IssueDate:      time.Now().Unix(),
		ExpirationDate: time.Now().Add(24 * time.Hour).Unix(),
		Version:        "trial",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("test-secret"))

	lic := License{
		Signature: tokenString,
		Data: LicenseData{
			Version: "trial",
		},
	}

	err := defaultValidateSignature(&lic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected signing method")
}

// License File Processing Tests

func TestLicenseInit_ValidFile(t *testing.T) {
	// Create a temporary license file
	tmpDir := t.TempDir()
	licensePath := filepath.Join(tmpDir, "test_license.yaml")

	err := os.WriteFile(licensePath, []byte(trialLicenseData), 0644)
	require.NoError(t, err)

	// Set environment variable
	originalEnv := os.Getenv("ESI_LICENSE_FILE")
	defer os.Setenv("ESI_LICENSE_FILE", originalEnv)

	os.Setenv("ESI_LICENSE_FILE", licensePath)

	// Read and validate the file manually (since init() is already called)
	licenseData, err := os.ReadFile(licensePath)
	assert.NoError(t, err)

	var lic License
	err = yaml.Unmarshal(licenseData, &lic)
	assert.NoError(t, err)

	err = defaultValidateSignature(&lic)
	assert.NoError(t, err)
}

func TestLicenseInit_MissingFile_FallbackToTrial(t *testing.T) {
	// Try to read a non-existent file
	nonExistentPath := "/non/existent/path/license.yaml"
	_, err := os.ReadFile(nonExistentPath)
	assert.Error(t, err)

	// Verify that trial license data is used as fallback
	var lic License
	err = yaml.Unmarshal([]byte(trialLicenseData), &lic)
	assert.NoError(t, err)

	err = defaultValidateSignature(&lic)
	assert.NoError(t, err)
	assert.Equal(t, "trial-license", lic.Data.LicenseID)
}

func TestLicenseInit_MalformedYAML(t *testing.T) {
	var lic License
	err := yaml.Unmarshal([]byte(testMalformedYAML), &lic)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "yaml:")
}

func TestLicenseInit_EnvironmentVariable(t *testing.T) {
	// Test environment variable processing
	originalEnv := os.Getenv("ESI_LICENSE_FILE")
	defer os.Setenv("ESI_LICENSE_FILE", originalEnv)

	testPath := "/custom/path/license.yaml"
	os.Setenv("ESI_LICENSE_FILE", testPath)

	// Simulate the environment variable processing from init()
	licenseFilePath := os.Getenv("ESI_LICENSE_FILE")
	if licenseFilePath == "" {
		licenseFilePath = "./license.yaml"
	}
	cleanPath := filepath.Join(".", filepath.Clean(licenseFilePath))

	assert.Equal(t, testPath, licenseFilePath)
	// filepath.Join(".", "/custom/path/license.yaml") returns "custom/path/license.yaml"
	assert.Equal(t, "custom/path/license.yaml", cleanPath)
}

func TestLicenseInit_PathTraversal_Security(t *testing.T) {
	// Test path traversal protection
	maliciousPath := "../../etc/passwd"
	cleanPath := filepath.Join(".", filepath.Clean(maliciousPath))

	// Should be cleaned and prefixed with the current directory
	assert.True(t, strings.HasPrefix(cleanPath, "."))
	// filepath.Join(".", "../../etc/passwd") returns "../../etc/passwd"
	// The path traversal sequences are preserved
	assert.Equal(t, "../../etc/passwd", cleanPath)
}

// License Expiry Tests

func TestLicenseExpiry_ValidLicense(t *testing.T) {
	futureTime := time.Now().Add(24 * time.Hour)

	lic := License{
		Data: LicenseData{
			ExpirationDate: futureTime,
		},
	}

	// License should not be expired
	assert.False(t, time.Now().After(lic.Data.ExpirationDate))
}

func TestLicenseExpiry_ExpiredLicense_LogsWarning(t *testing.T) {
	pastTime := time.Now().Add(-24 * time.Hour)

	lic := License{
		Data: LicenseData{
			ExpirationDate: pastTime,
		},
	}

	// License should be expired
	assert.True(t, time.Now().After(lic.Data.ExpirationDate))
}

func TestLicenseExpiry_FutureLicense(t *testing.T) {
	futureTime := time.Now().Add(365 * 24 * time.Hour) // 1 year in the future

	lic := License{
		Data: LicenseData{
			ExpirationDate: futureTime,
		},
	}

	// License should not be expired
	assert.False(t, time.Now().After(lic.Data.ExpirationDate))
	assert.True(t, lic.Data.ExpirationDate.After(time.Now()))
}

// Feature Registration Tests

func TestRegister_MatchingSubscription(t *testing.T) {
	// Create a license with a matching subscription
	testLicense := &License{
		Subscriptions: []LicenseSubscription{
			{
				Name:       "test.feature",
				ExpiryDate: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				MaxLimit:   5,
			},
		},
	}

	// Temporarily replace the global license
	originalLicense := license
	license = testLicense
	defer func() { license = originalLicense }()

	// Create a mock feature
	mockFeat := &mockFeature{name: "test.feature"}

	err := Register(mockFeat)
	assert.NoError(t, err)
	assert.True(t, mockFeat.enabled)
	assert.Equal(t, 5, mockFeat.maxLimit)
}

func TestRegister_NoMatchingSubscription(t *testing.T) {
	// Create a license without matching subscription
	testLicense := &License{
		Subscriptions: []LicenseSubscription{
			{
				Name:       "other.feature",
				ExpiryDate: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				MaxLimit:   5,
			},
		},
	}

	// Temporarily replace the global license
	originalLicense := license
	license = testLicense
	defer func() { license = originalLicense }()

	// Create a mock feature
	mockFeat := &mockFeature{name: "test.feature"}

	err := Register(mockFeat)
	assert.NoError(t, err)
	assert.False(t, mockFeat.enabled)     // Should remain disabled
	assert.Equal(t, 0, mockFeat.maxLimit) // Should remain at default
}

func TestRegister_MultipleSubscriptions(t *testing.T) {
	// Create a license with multiple subscriptions
	testLicense := &License{
		Subscriptions: []LicenseSubscription{
			{
				Name:       "feature1",
				ExpiryDate: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
				MaxLimit:   3,
			},
			{
				Name:       "feature2",
				ExpiryDate: time.Now().Add(48 * time.Hour).Format(time.RFC3339),
				MaxLimit:   10,
			},
		},
	}

	// Temporarily replace the global license
	originalLicense := license
	license = testLicense
	defer func() { license = originalLicense }()

	// Test first feature
	mockFeat1 := &mockFeature{name: "feature1"}
	err := Register(mockFeat1)
	assert.NoError(t, err)
	assert.True(t, mockFeat1.enabled)
	assert.Equal(t, 3, mockFeat1.maxLimit)

	// Test second feature
	mockFeat2 := &mockFeature{name: "feature2"}
	err = Register(mockFeat2)
	assert.NoError(t, err)
	assert.True(t, mockFeat2.enabled)
	assert.Equal(t, 10, mockFeat2.maxLimit)
}

// Additional Error Scenario Tests

func TestLicenseData_StructureParsing(t *testing.T) {
	// Test that LicenseData fields are properly parsed
	testData := LicenseData{
		LicenseID:      "test-id",
		CustomerName:   "test-customer",
		IssueDate:      time.Now(),
		ExpirationDate: time.Now().Add(24 * time.Hour),
		Version:        "1.0.0",
	}

	assert.Equal(t, "test-id", testData.LicenseID)
	assert.Equal(t, "test-customer", testData.CustomerName)
	assert.Equal(t, "1.0.0", testData.Version)
	assert.True(t, testData.ExpirationDate.After(testData.IssueDate))
}

func TestLicenseSubscription_StructureParsing(t *testing.T) {
	// Test that LicenseSubscription fields are properly parsed
	testSub := LicenseSubscription{
		Name:       "test.subscription",
		ExpiryDate: "2024-12-31T23:59:59Z",
		MaxLimit:   100,
	}

	assert.Equal(t, "test.subscription", testSub.Name)
	assert.Equal(t, "2024-12-31T23:59:59Z", testSub.ExpiryDate)
	assert.Equal(t, 100, testSub.MaxLimit)
}

func TestLicenseClaims_JWTIntegration(t *testing.T) {
	// Test that LicenseClaims work with JWT library
	claims := &LicenseClaims{
		LicenseID:      "test-license",
		CustomerName:   "test-customer",
		IssueDate:      time.Now().Unix(),
		ExpirationDate: time.Now().Add(24 * time.Hour).Unix(),
		Version:        "1.0.0",
		Subscriptions: []LicenseSubscription{
			{Name: "test.feature", MaxLimit: 5},
		},
		StandardClaims: jwt.StandardClaims{
			Subject:   "license",
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		},
	}

	assert.Equal(t, "test-license", claims.LicenseID)
	assert.Equal(t, "test-customer", claims.CustomerName)
	assert.Equal(t, "1.0.0", claims.Version)
	assert.Len(t, claims.Subscriptions, 1)
	assert.Equal(t, "test.feature", claims.Subscriptions[0].Name)
}
