package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the simulator configuration
type Config struct {
	// pumpX2 configuration
	PumpX2Path    string
	PumpX2Mode    string // "gradle" or "jar"
	PumpX2JarPath string // path to a prebuilt cliparser jar; if set, skips gradle entirely
	GradleCmd     string
	JavaCmd       string

	// JPAKE configuration
	JPAKEMode string // "go" or "pumpx2"

	// Logging configuration
	LogLevel string
}

// New creates a new configuration
func New(pumpX2Path, pumpX2Mode, jpakeMode, gradleCmd, javaCmd, logLevel, pumpX2JarPath string) (*Config, error) {
	// A prebuilt jar needs neither a pumpX2 checkout nor gradle, so skip all of
	// that validation and force jar mode when one is given.
	if pumpX2JarPath != "" {
		if _, err := os.Stat(pumpX2JarPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("pumpx2-jar-path does not exist: %s", pumpX2JarPath)
		}
		pumpX2Mode = "jar"
	} else {
		// Check for environment variable if path not provided
		if pumpX2Path == "" {
			pumpX2Path = os.Getenv("PUMPX2_PATH")
		}

		if pumpX2Path == "" {
			return nil, fmt.Errorf("pumpX2 path is required (use -pumpx2-path flag, -pumpx2-jar-path flag, or PUMPX2_PATH environment variable)")
		}

		// Validate that the path exists
		if _, err := os.Stat(pumpX2Path); os.IsNotExist(err) {
			return nil, fmt.Errorf("pumpX2 path does not exist: %s", pumpX2Path)
		}

		// Validate that it looks like a pumpX2 repository
		cliparserPath := filepath.Join(pumpX2Path, "cliparser")
		if _, err := os.Stat(cliparserPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not appear to be a pumpX2 repository (missing cliparser directory): %s", pumpX2Path)
		}

		gradlePath := filepath.Join(pumpX2Path, "gradlew")
		if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
			return nil, fmt.Errorf("path does not appear to be a pumpX2 repository (missing gradlew): %s", pumpX2Path)
		}
	}

	// Validate mode
	if pumpX2Mode != "gradle" && pumpX2Mode != "jar" {
		return nil, fmt.Errorf("invalid pumpx2-mode: %s (must be 'gradle' or 'jar')", pumpX2Mode)
	}

	// Validate JPAKE mode
	if jpakeMode != "go" && jpakeMode != "pumpx2" {
		return nil, fmt.Errorf("invalid jpake-mode: %s (must be 'go' or 'pumpx2')", jpakeMode)
	}

	return &Config{
		PumpX2Path:    pumpX2Path,
		PumpX2Mode:    pumpX2Mode,
		PumpX2JarPath: pumpX2JarPath,
		JPAKEMode:     jpakeMode,
		GradleCmd:     gradleCmd,
		JavaCmd:       javaCmd,
		LogLevel:      logLevel,
	}, nil
}
