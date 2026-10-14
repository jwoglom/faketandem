package config

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// Message codecs: which implementation of the pumpX2 message library parses
// requests and encodes responses (see pumpx2.Codec*).
const (
	MessageCodecNative  = "native"
	MessageCodecPumpX2  = "pumpx2"
	MessageCodecCompare = "compare"
)

// JPAKE modes.
const (
	JPAKEModeGo     = "go"
	JPAKEModePumpX2 = "pumpx2"
)

// Config holds the simulator configuration
type Config struct {
	// MessageCodec is MessageCodecNative unless pumpX2's cliparser was asked for.
	MessageCodec string

	// pumpX2 configuration, used only when the message codec or the JPAKE mode
	// names pumpX2.
	PumpX2Path    string
	PumpX2Mode    string // "gradle" or "jar"
	PumpX2JarPath string // path to a prebuilt cliparser jar; if set, skips gradle entirely
	GradleCmd     string
	JavaCmd       string

	// JPAKE configuration
	JPAKEMode        string // "go" or "pumpx2"
	JPAKELongTermKey []byte // pre-seeded long-term key for quick-pair reconnects, if provided

	// Logging configuration
	LogLevel string
}

// Options are the command-line settings New validates.
type Options struct {
	MessageCodec        string
	PumpX2Path          string
	PumpX2Mode          string
	PumpX2JarPath       string
	JPAKEMode           string
	GradleCmd           string
	JavaCmd             string
	LogLevel            string
	JPAKELongTermKeyHex string
}

// UsesPumpX2 reports whether anything in this configuration runs pumpX2.
func (c *Config) UsesPumpX2() bool {
	return c.MessageCodec != MessageCodecNative || c.JPAKEMode == JPAKEModePumpX2
}

// New validates the options. pumpX2 is checked for only when the message
// codec or the JPAKE mode asks for it; by default nothing needs it.
func New(o Options) (*Config, error) {
	if o.MessageCodec == "" {
		o.MessageCodec = MessageCodecNative
	}
	if o.JPAKEMode == "" {
		o.JPAKEMode = JPAKEModeGo
	}
	switch o.MessageCodec {
	case MessageCodecNative, MessageCodecPumpX2, MessageCodecCompare:
	default:
		return nil, fmt.Errorf("invalid message-codec: %s (must be 'native', 'pumpx2' or 'compare')", o.MessageCodec)
	}
	if o.JPAKEMode != JPAKEModeGo && o.JPAKEMode != JPAKEModePumpX2 {
		return nil, fmt.Errorf("invalid jpake-mode: %s (must be 'go' or 'pumpx2')", o.JPAKEMode)
	}

	cfg := &Config{
		MessageCodec: o.MessageCodec,
		PumpX2Mode:   o.PumpX2Mode,
		JPAKEMode:    o.JPAKEMode,
		GradleCmd:    o.GradleCmd,
		JavaCmd:      o.JavaCmd,
		LogLevel:     o.LogLevel,
	}
	if cfg.UsesPumpX2() {
		if err := cfg.resolvePumpX2(o.PumpX2Path, o.PumpX2JarPath); err != nil {
			return nil, err
		}
	}

	if o.JPAKELongTermKeyHex != "" {
		key, err := hex.DecodeString(o.JPAKELongTermKeyHex)
		if err != nil {
			return nil, fmt.Errorf("invalid jpake-long-term-key (must be hex-encoded): %w", err)
		}
		cfg.JPAKELongTermKey = key
	}
	return cfg, nil
}

// resolvePumpX2 finds the cliparser: a prebuilt jar, or a pumpX2 checkout to
// run it from.
func (c *Config) resolvePumpX2(pumpX2Path, jarPath string) error {
	// A prebuilt jar needs neither a pumpX2 checkout nor gradle, so skip all of
	// that validation and force jar mode when one is given.
	if jarPath != "" {
		if _, err := os.Stat(jarPath); os.IsNotExist(err) {
			return fmt.Errorf("pumpx2-jar-path does not exist: %s", jarPath)
		}
		c.PumpX2JarPath = jarPath
		c.PumpX2Mode = "jar"
		return nil
	}

	if pumpX2Path == "" {
		pumpX2Path = os.Getenv("PUMPX2_PATH")
	}
	if pumpX2Path == "" {
		return fmt.Errorf("message-codec %q / jpake-mode %q need pumpX2: set -pumpx2-jar-path, -pumpx2-path or PUMPX2_PATH (or use the defaults, which need neither)",
			c.MessageCodec, c.JPAKEMode)
	}
	if _, err := os.Stat(pumpX2Path); os.IsNotExist(err) {
		return fmt.Errorf("pumpX2 path does not exist: %s", pumpX2Path)
	}
	if _, err := os.Stat(filepath.Join(pumpX2Path, "cliparser")); os.IsNotExist(err) {
		return fmt.Errorf("path does not appear to be a pumpX2 repository (missing cliparser directory): %s", pumpX2Path)
	}
	if _, err := os.Stat(filepath.Join(pumpX2Path, "gradlew")); os.IsNotExist(err) {
		return fmt.Errorf("path does not appear to be a pumpX2 repository (missing gradlew): %s", pumpX2Path)
	}
	if c.PumpX2Mode != "gradle" && c.PumpX2Mode != "jar" {
		return fmt.Errorf("invalid pumpx2-mode: %s (must be 'gradle' or 'jar')", c.PumpX2Mode)
	}
	c.PumpX2Path = pumpX2Path
	return nil
}
