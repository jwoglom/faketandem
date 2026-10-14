package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCurrentEGVGuiDataResponse(t *testing.T) {
	for _, tc := range []struct {
		name      string
		hex       string
		txID      int
		timestamp int64
		reading   int32
		status    currentstatus.EGVStatus
		trendRate int32
	}{
		{"no CGM", "000323030800000000000004007b52", 3, 0, 0, currentstatus.EGVStatusUnavailable, 0},
		{"123 falling 1", "0032233208c87c131c7b0001ff5eb5", 50, 471039176, 123, currentstatus.EGVStatusValid, -1},
		{"122 falling 1", "004c234c08f27d131c7a0001ff5e4a", 76, 471039474, 122, currentstatus.EGVStatusValid, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				currentstatus.NewCurrentEGVGuiDataResponse(tc.timestamp, tc.reading, tc.status.ID(), tc.trendRate)).(*currentstatus.CurrentEGVGuiDataResponse)
			if parsed.EgvStatus == nil || *parsed.EgvStatus != tc.status {
				t.Errorf("status %v, want %s", parsed.EgvStatus, tc.status)
			}
		})
	}
}

func TestCurrentEGVGuiDataResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CurrentEGVGuiDataResponse")
}
