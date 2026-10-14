package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStopTempRateResponse(t *testing.T) {
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1906112"}}
	for _, tc := range []struct {
		name       string
		tempRateID byte
		hex        string
		txID       int
	}{
		{"temp rate 9", 9, "002ca72c1b000900d9448b1ea1dcb10f6cd0a00465d9091bad121938e17e51d4c3c9", 44},
		{"temp rate 15", 15, "0051a7511b000f00caa3ee1f206002d65b54e77f42869917c881cae3dde6ad621928", 81},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := control.NewStopTempRateResponseFromCargo([]byte{0, tc.tempRateID, 0})
			if err != nil {
				t.Fatal(err)
			}
			tester.Test(t, tc.hex, tc.txID, 1, messages.Control, expected)
		})
	}
}

func TestStopTempRateResponseFromAStatusAndID(t *testing.T) {
	parsed := parity.TestMessage(t, "007ca77c1b00610028034f20de866872a5759f0a90733baae767381b6184106f17ed", 124, 1, messages.Control,
		control.NewStopTempRateResponse(0, 97)).(*control.StopTempRateResponse)
	if parsed.Status != 0 || parsed.TempRateID != 97 {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestStopTempRateResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StopTempRateResponse")
}
