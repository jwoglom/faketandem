package authentication_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJpake4KeyConfirmationResponse(t *testing.T) {
	expected, err := authentication.NewJpake4KeyConfirmationResponse(0, decodeHex(t, "e3fd32509dfacf47"), make([]byte, 8),
		decodeHex(t, "d0410856c350ce6b9756e03810c6f99a4a0743160a1fdb0973db2f90bdc9a96a"))
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "00042904320000e3fd32509dfacf470000000000000000d0410856c350ce6b9756e03810c6f99a4a0743160a1fdb0973db2f90bdc9a96ad742",
		4, 4, messages.Authorization, expected).(*authentication.Jpake4KeyConfirmationResponse)
	if parsed.AppInstanceID != 0 || !bytes.Equal(parsed.Nonce, expected.Nonce) || !bytes.Equal(parsed.Reserved, expected.Reserved) ||
		!bytes.Equal(parsed.HashDigest, expected.HashDigest) {
		t.Errorf("parsed %+v, want %+v", parsed, expected)
	}
}

func TestJpake4KeyConfirmationResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "Jpake4KeyConfirmationResponse")
}
