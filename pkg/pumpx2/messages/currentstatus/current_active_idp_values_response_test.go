package currentstatus_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// The target BG reads 11264, not 115: pumpX2 reads it a byte late.
func TestCurrentActiveIdpValuesResponse(t *testing.T) {
	raw, _ := hex.DecodeString("7017000073002c012800")
	parsed := &currentstatus.CurrentActiveIdpValuesResponse{}
	if err := parsed.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), raw) {
		t.Errorf("cargo %x", parsed.Cargo())
	}
	if parsed.CurrentCarbRatio != 6000 || parsed.CurrentTargetBg != 11264 || parsed.CurrentInsulinDuration != 300 || parsed.CurrentIsf != 40 {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestCurrentActiveIdpValuesResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentActiveIdpValuesResponse")
}
