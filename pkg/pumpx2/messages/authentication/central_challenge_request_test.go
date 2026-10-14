package authentication_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func decodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCentralChallengeRequest(t *testing.T) {
	expected, err := authentication.NewCentralChallengeRequest(0, decodeHex(t, "4d08435da2694735"))
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "000010000a00004d08435da26947356d6f", 0, 1, messages.Authorization, expected).(*authentication.CentralChallengeRequest)
	if parsed.AppInstanceID != expected.AppInstanceID || !bytes.Equal(parsed.CentralChallenge, expected.CentralChallenge) {
		t.Errorf("parsed %+v, want %+v", parsed, expected)
	}
}

func TestCentralChallengeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CentralChallengeRequest")
}
