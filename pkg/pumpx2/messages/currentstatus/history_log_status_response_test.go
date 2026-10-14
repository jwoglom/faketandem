package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestHistoryLogStatusResponse(t *testing.T) {
	parity.TestMessage(t, "00033b030cc40a000000000000c30a0000d84d", 3, 1, messages.CurrentStatus,
		currentstatus.NewHistoryLogStatusResponse(2756, 0, 2755))
}

func TestHistoryLogStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "HistoryLogStatusResponse")
}
