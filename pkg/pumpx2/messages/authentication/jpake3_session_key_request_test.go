package authentication_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJpake3SessionKeyRequest(t *testing.T) {
	expected := authentication.NewJpake3SessionKeyRequestFromCargo([]byte{0, 0})
	parsed := parity.TestMessage(t, "000326030200008121", 3, 1, messages.Authorization, expected).(*authentication.Jpake3SessionKeyRequest)
	if parsed.ChallengeParam != 0 {
		t.Errorf("challenge param %d, want 0", parsed.ChallengeParam)
	}
}

func TestJpake3SessionKeyRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "Jpake3SessionKeyRequest")
}
