package authentication_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpChallengeRequest(t *testing.T) {
	expected, err := authentication.NewPumpChallengeRequest(1, decodeHex(t, "0194a8f98ca49cddf70c2c1331730290bca3df07"))
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessagePackets(t, []string{"010112011601000194a8f98ca49cddf70c2c1331", "0001730290bca3df079967"},
		1, messages.Authorization, expected).(*authentication.PumpChallengeRequest)
	if parsed.AppInstanceID != 1 || !bytes.Equal(parsed.PumpChallengeHash, expected.PumpChallengeHash) {
		t.Errorf("parsed %+v, want %+v", parsed, expected)
	}
}

func TestPumpChallengeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpChallengeRequest")
}
