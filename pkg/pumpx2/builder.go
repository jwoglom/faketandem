package pumpx2

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	// Cache for JAR path
	jarPathCache     string
	jarPathCacheLock sync.Mutex
)

// BuildCliParserJAR builds the cliparser JAR if it doesn't exist
func BuildCliParserJAR(pumpX2Path, gradleCmd string) (string, error) {
	jarPathCacheLock.Lock()
	defer jarPathCacheLock.Unlock()

	// Check cache first
	if jarPathCache != "" {
		if _, err := os.Stat(jarPathCache); err == nil {
			log.Debugf("Using cached cliparser JAR: %s", jarPathCache)
			return jarPathCache, nil
		}
		// Cache is stale, clear it
		jarPathCache = ""
	}

	// Expected JAR location
	jarPath := filepath.Join(pumpX2Path, "cliparser", "build", "libs", "pumpx2-cliparser-all.jar")

	// Check if JAR already exists
	if _, err := os.Stat(jarPath); err == nil {
		log.Infof("Found existing cliparser JAR: %s", jarPath)
		jarPathCache = jarPath
		return jarPath, nil
	}

	// JAR doesn't exist, need to build it
	log.Info("cliparser JAR not found, building via gradle...")
	log.Info("This may take a few minutes on first run...")

	// Determine gradle command to use
	gradlePath := filepath.Join(pumpX2Path, gradleCmd)
	if _, err := os.Stat(gradlePath); os.IsNotExist(err) {
		// Try without path
		gradlePath = gradleCmd
	}

	// Build the JAR
	cmd := exec.Command(gradlePath, ":cliparser:shadowJar")
	cmd.Dir = pumpX2Path
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	start := time.Now()
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build cliparser JAR: %w", err)
	}
	duration := time.Since(start)

	log.Infof("Built cliparser JAR in %.1f seconds", duration.Seconds())

	// Verify JAR was created
	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		return "", fmt.Errorf("JAR build appeared to succeed but file not found at: %s", jarPath)
	}

	jarPathCache = jarPath
	return jarPath, nil
}
