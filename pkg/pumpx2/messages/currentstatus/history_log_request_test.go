package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestHistoryLogRequest(t *testing.T) {
	parity.TestMessage(t, "00043c04056e0f00000a4247", 4, 1, messages.CurrentStatus, currentstatus.NewHistoryLogRequest(3950, 10))
}

func TestHistoryLogRequestForAFullSliceReadsTheCountSignedButCountsAll(t *testing.T) {
	parsed := &currentstatus.HistoryLogRequest{}
	if err := parsed.Parse(currentstatus.NewHistoryLogRequest(300, 255).Cargo()); err != nil {
		t.Fatal(err)
	}
	if parsed.StartLog != 300 || parsed.NumberOfLogs != -1 || parsed.LogCount() != 255 {
		t.Errorf("start %d, number of logs %d, count %d", parsed.StartLog, parsed.NumberOfLogs, parsed.LogCount())
	}
}

func TestHistoryLogRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "HistoryLogRequest")
}
