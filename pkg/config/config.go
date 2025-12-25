package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the simulator configuration
type Config struct {
	// pumpX2 configuration
	PumpX2Path string
	PumpX2Mode string // "gradle" or "jar"
	GradleCmd  string
	JavaCmd    string

	// Logging configuration
	LogLevel string
}

// New creates a new configuration
func New(pumpX2Path, pumpX2Mode, gradleCmd, javaCmd, logLevel string) (*Config, error) {
	// Check for environment variable if path not provided
	if pumpX2Path == "" {
		pumpX2Path = os.Getenv("PUMPX2_PATH")
	}

	if pumpX2Path == "" {
		return nil, fmt.Errorf("pumpX2 path is required (use -pumpx2-path flag or PUMPX2_PATH environment variable)")
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

	// Validate mode
	if pumpX2Mode != "gradle" && pumpX2Mode != "jar" {
		return nil, fmt.Errorf("invalid pumpx2-mode: %s (must be 'gradle' or 'jar')", pumpX2Mode)
	}

	return &Config{
		PumpX2Path: pumpX2Path,
		PumpX2Mode: pumpX2Mode,
		GradleCmd:  gradleCmd,
		JavaCmd:    javaCmd,
		LogLevel:   logLevel,
	}, nil
}
