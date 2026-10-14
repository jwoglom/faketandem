package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestNonControlIQIOBResponse(t *testing.T) {
	parity.TestMessage(t, "000427040cc300000030390000f100000008f7", 4, 1, messages.CurrentStatus,
		currentstatus.NewNonControlIQIOBResponse(195, 14640, 241))
}

func TestNonControlIQIOBResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "NonControlIQIOBResponse")
}
