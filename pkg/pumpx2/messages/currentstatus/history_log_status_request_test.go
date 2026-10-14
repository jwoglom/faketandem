package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestHistoryLogStatusRequest(t *testing.T) {
	parity.TestMessage(t, "00033a0300ab9b", 3, 1, messages.CurrentStatus, currentstatus.NewHistoryLogStatusRequest())
}

func TestHistoryLogStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "HistoryLogStatusRequest")
}
