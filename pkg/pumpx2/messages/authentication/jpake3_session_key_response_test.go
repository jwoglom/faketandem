package authentication_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/authentication"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJpake3SessionKeyResponse(t *testing.T) {
	fromCargo, err := authentication.NewJpake3SessionKeyResponseFromCargo(decodeHex(t, "000002e91fff505adb4f0000000000000000"))
	if err != nil {
		t.Fatal(err)
	}
	fromFields, err := authentication.NewJpake3SessionKeyResponse(0, decodeHex(t, "02e91fff505adb4f"), make([]byte, 8))
	if err != nil {
		t.Fatal(err)
	}
	for name, expected := range map[string]*authentication.Jpake3SessionKeyResponse{"from cargo": fromCargo, "from fields": fromFields} {
		t.Run(name, func(t *testing.T) {
			parsed := parity.TestMessage(t, "0003270312000002e91fff505adb4f0000000000000000ba54", 3, 2, messages.Authorization, expected).(*authentication.Jpake3SessionKeyResponse)
			if parsed.AppInstanceID != expected.AppInstanceID || !bytes.Equal(parsed.DeviceKeyNonce, expected.DeviceKeyNonce) ||
				!bytes.Equal(parsed.DeviceKeyReserved, expected.DeviceKeyReserved) {
				t.Errorf("parsed %+v, want %+v", parsed, expected)
			}
		})
	}
}

func TestJpake3SessionKeyResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "Jpake3SessionKeyResponse")
}
