package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestHistoryLogResponse(t *testing.T) {
	parsed := parity.TestMessage(t, "00033d0302000105d9", 3, 1, messages.CurrentStatus,
		currentstatus.NewHistoryLogResponse(0, 1)).(*currentstatus.HistoryLogResponse)
	if !parsed.IsStatusOK() || parsed.StreamID != 1 {
		t.Errorf("status %d, stream %d", parsed.Status, parsed.StreamID)
	}
}

func TestHistoryLogResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "HistoryLogResponse")
}
