package currentstatus_test

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLastBolusStatusV2ResponseEmpty(t *testing.T) {
	parity.TestMessage(t, "0004a504180000000000000000000000000000000000000000000000008521", 4, 2, messages.CurrentStatus,
		currentstatus.NewLastBolusStatusV2Response(0, 0, 0, 0, 0, 0, 0, 0, 0))
}

func TestLastBolusStatusV2Response(t *testing.T) {
	completed, stopped := currentstatus.BolusStatusComplete, currentstatus.BolusStatusStoppedUserTerminated
	for _, tc := range []struct {
		name                            string
		hex                             string
		txID                            int
		bolusID                         int32
		timestamp, delivered            int64
		statusID, sourceID, typeBitmask int32
		extendedDuration, requested     int64
		status                          currentstatus.BolusStatus
	}{
		{"completed", "0004a5041801a80c0000b453551b0e1a0000030101000000000e1a0000b898", 4, 3240, 458576820, 6670, 3, 1, 1, 0, 6670, completed},
		{"0.5u stopped", "0007a5071801aa0c0000bc79551b320000000001080000000032000000893b", 7, 3242, 458586556, 50, 0, 1, 8, 0, 50, stopped},
		{"0.5u completed", "0008a5081801ab0c0000347a551b3200000003010800000000320000001462", 8, 3243, 458586676, 50, 3, 1, 8, 0, 50, completed},
		{"partly delivered", "0004a5041801ad0c00008a99551bb707000000010800000000c4090000bc9e", 4, 3245, 458594698, 1975, 0, 1, 8, 0, 2500, stopped},
		// The extended part of a bolus still running reads as a duration of 0.
		{"extended, first part delivered", "0010a5101801b00c0000f29b551bdc05000003010800000000dc050000c546", 16, 3248, 458595314, 1500, 3, 1, 8, 0, 1500, completed},
		{"extended, stopped", "0016a5161801b00c00009a9c551b15000000000104a8000000f40100003714", 22, 3248, 458595482, 21, 0, 1, 4, 168, 500, stopped},
		{"quick bolus canceled once delivered", "0019a5191801b30c000041a3551bf401000000000100000000f4010000afcc", 25, 3251, 458597185, 500, 0, 0, 1, 0, 500, stopped},
		{"Control-IQ automatic bolus", "0004a5041801ca0c0000cd32581b310600000307020000000031060000a8ca", 4, 3274, 458765005, 1585, 3, 7, 2, 0, 1585, completed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := currentstatus.NewLastBolusStatusV2Response(1, tc.bolusID, tc.timestamp, tc.delivered, tc.statusID, tc.sourceID, tc.typeBitmask,
				tc.extendedDuration, tc.requested)
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 2, messages.CurrentStatus, expected).(*currentstatus.LastBolusStatusV2Response)
			if s := parsed.BolusStatus(); s == nil || *s != tc.status {
				t.Errorf("bolus status %v, want %v", s, tc.status)
			}
		})
	}
}

func TestLastBolusStatusV2ResponseTimestampIsWhenTheBolusEnded(t *testing.T) {
	m := currentstatus.NewLastBolusStatusV2Response(1, 3240, 458576820, 6670, 3, 1, 1, 0, 6670)
	want := time.Date(2022, time.July, 13, 14, 27, 0, 0, time.UTC)
	if at, err := m.TimestampInstant(); err != nil || !at.Equal(want) {
		t.Errorf("at %v (%v), want %v", at, err, want)
	}
}

func TestLastBolusStatusV2ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LastBolusStatusV2Response")
}
