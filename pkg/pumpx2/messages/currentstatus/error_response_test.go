package currentstatus_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestErrorResponse(t *testing.T) {
	for _, tc := range []struct {
		name        string
		hex         string
		txID        int
		requestCode int32
		errorCode   currentstatus.ErrorCode
	}{
		{"bad opcode", "00054d05029006168b", 5, -112, currentstatus.ErrorCodeBadOpcode},
		{"invalid parameter", "00084d08024207f4d1", 8, 66, currentstatus.ErrorCodeInvalidRequiredParameter},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				currentstatus.NewErrorResponse(tc.requestCode, tc.errorCode.ID())).(*currentstatus.ErrorResponse)
			if parsed.RequestCodeID != tc.requestCode || parsed.ErrorCode == nil || *parsed.ErrorCode != tc.errorCode || parsed.RemainingBytes != nil {
				t.Errorf("parsed %s", messages.ToString(parsed))
			}
		})
	}
}

func TestErrorResponseKeepsWhatFollowsASignedRequestsError(t *testing.T) {
	raw := append([]byte{0x9e, 0x09}, bytes.Repeat([]byte{0xaa}, 24)...)
	m := &currentstatus.ErrorResponse{}
	if err := m.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if m.ErrorCode == nil || *m.ErrorCode != currentstatus.ErrorCodeInvalidAuthenticationError || !bytes.Equal(m.RemainingBytes, raw[2:]) {
		t.Errorf("parsed %s", messages.ToString(m))
	}
}

func TestErrorResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ErrorResponse")
}
