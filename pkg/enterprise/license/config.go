// Copyright External Secrets Inc. 2025
//
//	All Rights Reserved
package license

// Config holds the configuration for the license service.
type Config struct {
	// LicenseFilePath is the path to the license file
	LicenseFilePath string

	// StrictMode determines whether the application should fail to start if license validation fails
	StrictMode bool
}

// NewConfig creates a new license service configuration.
func NewConfig(licenseFilePath string, strictMode bool) *Config {
	return &Config{
		LicenseFilePath: licenseFilePath,
		StrictMode:      strictMode,
	}
}
