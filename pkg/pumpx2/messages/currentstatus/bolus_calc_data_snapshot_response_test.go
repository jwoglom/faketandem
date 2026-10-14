package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBolusCalcDataSnapshotResponse(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		hex                      string
		txID                     int
		correctionFactor         int32
		iob, maxBolusHourlyTotal int64
		unknown11bytes           []byte
	}{
		{"1", "000373032e00d100790b000078006e001e000170170000a861fa0a0000000090a9541bdd0f000001fb0101555555555555555571b3",
			3, 209, 2937, 2810, []byte{0x90, 0xa9, 0x54, 0x1b, 0xdd, 0x0f, 0, 0, 1, 0xfb, 1}},
		{"2", "000473042e00ae00fe08000078006e001e000170170000a86100000000000040ae541b8d14000001f001015555555555555555d2d7",
			4, 174, 2302, 0, []byte{0x40, 0xae, 0x54, 0x1b, 0x8d, 0x14, 0, 0, 1, 0xf0, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := currentstatus.NewBolusCalcDataSnapshotResponseWithUnknownBytes(false, tc.correctionFactor, tc.iob, 120, 110, 30, true, 6000, 25000,
				tc.maxBolusHourlyTotal, false, false, true, tc.unknown11bytes, []byte{0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55, 0x55})
			parity.TestMessage(t, tc.hex, tc.txID, 3, messages.CurrentStatus, expected)
		})
	}
}

func TestBolusCalcDataSnapshotResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BolusCalcDataSnapshotResponse")
}
