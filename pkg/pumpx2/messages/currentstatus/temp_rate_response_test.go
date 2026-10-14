package currentstatus_test

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestTempRateResponse(t *testing.T) {
	parity.TestMessage(t, "00032b030a0000000000000000000039c6", 3, 1, messages.CurrentStatus,
		currentstatus.NewTempRateResponse(false, 0, 0, 0))
}

func TestTempRateResponseReadsActiveTempRates(t *testing.T) {
	for _, tc := range []struct {
		cargo      []byte
		duration   int64
		startTime  int64
		percentage int32
	}{
		{[]byte{1, 0xfa, 0xda, 0, 0xf3, 0x1f, 0x84, 3, 0, 0}, 900, 536019162, 250},
		{[]byte{1, 0xf5, 0x9d, 5, 0xf3, 0x1f, 0x84, 3, 0, 0}, 900, 536020381, 245},
		{[]byte{1, 0xc8, 0x39, 5, 0xf3, 0x1f, 8, 7, 0, 0}, 1800, 536020281, 200},
	} {
		r, err := currentstatus.NewTempRateResponseFromCargo(tc.cargo)
		if err != nil {
			t.Fatal(err)
		}
		if !r.Active || r.Duration != tc.duration || r.StartTimeRaw != tc.startTime || r.Percentage != tc.percentage {
			t.Errorf("%x: %+v, want active, %d%% from %d for %d", tc.cargo, *r, tc.percentage, tc.startTime, tc.duration)
		}
	}
}

func TestTempRateResponseStartTimeIsOnThePumpEpoch(t *testing.T) {
	start, err := currentstatus.NewTempRateResponse(true, 250, 536019162, 900).StartTimeInstant()
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2024, 12, 25, 22, 12, 42, 0, time.UTC); !start.Equal(want) {
		t.Errorf("start %v, want %v", start, want)
	}
}

func TestTempRateResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "TempRateResponse")
}
