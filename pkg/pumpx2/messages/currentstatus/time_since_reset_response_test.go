package currentstatus_test

import (
	"math"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestTimeSinceResetResponse(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		hex                             string
		txID                            int
		currentTime, pumpTimeSinceReset int64
	}{
		{"Feb 28, 2022 5:50:49pm", "0003370308f988a31a69290000db69", 3, 446925049, 10601},
		{"388 seconds later", "00043704087d8aa31aed2a0000b7a9", 4, 446925437, 10989},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := currentstatus.NewTimeSinceResetResponse(tc.currentTime, tc.pumpTimeSinceReset)
			if err != nil {
				t.Fatal(err)
			}
			parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus, expected)
		})
	}
}

func TestTimeSinceResetResponseCurrentTimeIsOnThePumpEpoch(t *testing.T) {
	m := &currentstatus.TimeSinceResetResponse{}
	if err := m.Parse([]byte{0xf9, 0x88, 0xa3, 0x1a, 0x69, 0x29, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2022, 2, 28, 17, 50, 49, 0, time.UTC); !m.CurrentTimeInstant.Equal(want) {
		t.Errorf("current time %v, want %v", m.CurrentTimeInstant, want)
	}
}

func TestTimeSinceResetResponseRefusesATimeBeyondJavasInstant(t *testing.T) {
	if _, err := currentstatus.NewTimeSinceResetResponse(math.MaxInt64/2, 0); err == nil {
		t.Error("a current time past Instant.MAX was built")
	}
}

func TestTimeSinceResetResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "TimeSinceResetResponse")
}
