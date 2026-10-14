package messages

import (
	"os"
	"strconv"
	"strings"
)

// ignoreInvalidHmac is PacketArrayList.IGNORE_INVALID_HMAC, the pairing code
// cliparser falls back to when none is configured: it makes a parse log a bad
// CRC or signature rather than throw.
const ignoreInvalidHmac = "IGNORE_HMAC_SIGNATURE_EXCEPTION"

// Env is the environment a cliparser command reads, which faketandem's
// serve runner forwards from its own process environment.
type Env struct {
	AuthenticationKey string // PUMP_AUTHENTICATION_KEY
	PairingCode       string // PUMP_PAIRING_CODE
	TimeSinceReset    string // PUMP_TIME_SINCE_RESET
	MaxChunkSize      string // PUMPX2_MAX_CHUNK_SIZE
}

// EnvFromOS reads the same variables the cliparser serve runner forwards.
func EnvFromOS() Env {
	return Env{
		AuthenticationKey: os.Getenv("PUMP_AUTHENTICATION_KEY"),
		PairingCode:       os.Getenv("PUMP_PAIRING_CODE"),
		TimeSinceReset:    os.Getenv("PUMP_TIME_SINCE_RESET"),
		MaxChunkSize:      os.Getenv("PUMPX2_MAX_CHUNK_SIZE"),
	}
}

func isBlank(s string) bool { return strings.TrimSpace(s) == "" }

// authKey is PumpStateSupplier.determinePumpAuthKey as cliparser configures
// it: with no JPAKE secret, the pairing code's own bytes.
func (e Env) authKey() []byte {
	switch {
	case !isBlank(e.PairingCode):
		return []byte(e.PairingCode)
	case !isBlank(e.AuthenticationKey):
		return []byte(e.AuthenticationKey)
	default:
		return []byte(ignoreInvalidHmac)
	}
}

// timeSinceReset is Long.valueOf(PUMP_TIME_SINCE_RESET), or 0 when blank.
func (e Env) timeSinceReset() (int64, error) {
	if isBlank(e.TimeSinceReset) {
		return 0, nil
	}
	v, err := strconv.ParseInt(e.TimeSinceReset, 10, 64)
	if err != nil {
		return 0, exception("java.lang.NumberFormatException", "For input string: %q", e.TimeSinceReset)
	}
	return v, nil
}

// shouldIgnoreInvalidHmac is PacketArrayList.shouldIgnoreInvalidHmac.
func shouldIgnoreInvalidHmac(key []byte) bool {
	n := len(ignoreInvalidHmac)
	padded := make([]byte, n)
	copy(padded, key)
	return string(padded) == ignoreInvalidHmac
}
