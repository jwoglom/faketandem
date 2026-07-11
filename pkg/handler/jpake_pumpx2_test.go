package handler

import (
	"os"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// TestEncodeClientRequest_ExcludesCargoField guards against a regression where
// encodeClientRequest passed cliparser's own "cargo" field (present on every
// parsed message's output, redundant with the message's other named fields)
// through to EncodeMessage. cliparser's "encode" command picks a constructor by
// matching parameter *count*, so the extra key made it fail to find one with
// "no constructor was found with 3 parameters" for any message re-encoded this
// way -- confirmed against a real cliparser jar. Skipped unless
// FAKETANDEM_TEST_CLIPARSER_JAR is set, since CI has no jar available.
func TestEncodeClientRequest_ExcludesCargoField(t *testing.T) {
	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping real jar integration test")
	}

	bridge, err := pumpx2.NewBridge("", "jar", "", "java", jarPath)
	if err != nil {
		t.Fatalf("failed to create bridge: %v", err)
	}

	auth := NewPumpX2JPAKEAuthenticator("123456", bridge, "", "jar", "", "java", jarPath)

	// Real centralChallenge bytes captured from an actual Jpake1aRequest, plus
	// the redundant "cargo" field that a real ParseMessage call would also
	// produce -- encodeClientRequest must strip it before calling EncodeMessage.
	requestData := map[string]interface{}{
		"messageName":      "Jpake1aRequest",
		"appInstanceId":    0,
		"centralChallenge": "41045483658e8ea056f5b4d1454c13740db3a9712830938ea074fb0096489f4d5a8a16fa09767adcbdc6e8f74550d91c5ebe9fa3a18f91c2e73d12e182a2cb60a64f41049da3799b6ba274f3a83ee4b8b4e456cd262292db6dd35f62b91843e1e418700c1a97be5d09e26bd5a11956ee6f4819c09f71f60522e1418aa0a9e6afb07390512011b80880ed77972d4435cdfb223a7f30c54bff805c1308796e36f5b468e62c1f",
		"cargo":            "000041045483658e8ea056f5b4d1454c13740db3a9712830938ea074fb0096489f4d5a8a16fa09767adcbdc6e8f74550d91c5ebe9fa3a18f91c2e73d12e182a2cb60a64f41049da3799b6ba274f3a83ee4b8b4e456cd262292db6dd35f62b91843e1e418700c1a97be5d09e26bd5a11956ee6f4819c09f71f60522e1418aa0a9e6afb07390512011b80880ed77972d4435cdfb223a7f30c54bff805c1308796e36f5b468e62c1f",
	}

	result := auth.encodeClientRequest(requestData)
	if result == "00" {
		t.Fatal("encodeClientRequest fell back to the \"00\" placeholder -- encoding failed (see logs)")
	}

	// A real round-trip re-encode of the exact fragments the phone sent should
	// reproduce the exact same bytes, other than txID (encodeClientRequest always
	// re-encodes with txID=0, since jpake-server doesn't validate it).
	const expectedFirstFragment = "09002000a7000041045483658e8ea056f5b4d145"
	if len(result) < len(expectedFirstFragment) || result[:len(expectedFirstFragment)] != expectedFirstFragment {
		t.Errorf("expected re-encoded output to start with %q, got %q", expectedFirstFragment, result)
	}
}
