package authentication_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCentralChallengeResponse(t *testing.T) {
	expected, err := authentication.NewCentralChallengeResponse(1, decodeHex(t, "8c212d7a8fbda85f83a3440254488dfb561264ec"), decodeHex(t, "840c4e16873046bc"))
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "000011001e01008c212d7a8fbda85f83a3440254488dfb561264ec840c4e16873046bc2c1a",
		0, 2, messages.Authorization, expected).(*authentication.CentralChallengeResponse)
	if parsed.AppInstanceID != 1 || !bytes.Equal(parsed.CentralChallengeHash, expected.CentralChallengeHash) || !bytes.Equal(parsed.HmacKey, expected.HmacKey) {
		t.Errorf("parsed %+v, want %+v", parsed, expected)
	}
}

func TestCentralChallengeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CentralChallengeResponse")
}
