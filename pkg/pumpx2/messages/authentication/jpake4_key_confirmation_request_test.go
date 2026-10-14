package authentication_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJpake4KeyConfirmationRequest(t *testing.T) {
	expected, err := authentication.NewJpake4KeyConfirmationRequest(0, decodeHex(t, "571ad034741c777d"), make([]byte, 8),
		decodeHex(t, "2fece0828a91f7372a47cb7bf597a296961f66f0f45c4b7b76e9aeaf8f176f6d"))
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "03042804320000571ad034741c777d0000000000", 4, 4, messages.Authorization, expected,
		"02040000002fece0828a91f7372a47cb7bf597a2",
		"010496961f66f0f45c4b7b76e9aeaf8f176f6d44",
		"00046e",
	).(*authentication.Jpake4KeyConfirmationRequest)
	if parsed.AppInstanceID != 0 || !bytes.Equal(parsed.Nonce, expected.Nonce) || !bytes.Equal(parsed.Reserved, expected.Reserved) ||
		!bytes.Equal(parsed.HashDigest, expected.HashDigest) {
		t.Errorf("parsed %+v, want %+v", parsed, expected)
	}
}

func TestJpake4KeyConfirmationRequestRefusesAShortDigest(t *testing.T) {
	if _, err := authentication.NewJpake4KeyConfirmationRequest(0, make([]byte, 8), make([]byte, 8), make([]byte, 31)); err == nil {
		t.Error("a 31-byte digest was accepted")
	}
}

func TestJpake4KeyConfirmationRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "Jpake4KeyConfirmationRequest")
}
