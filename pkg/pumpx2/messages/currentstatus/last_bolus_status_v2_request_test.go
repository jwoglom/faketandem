package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLastBolusStatusV2Request(t *testing.T) {
	parity.TestMessage(t, "0004a404000461", 4, 1, messages.CurrentStatus, currentstatus.NewLastBolusStatusV2Request())
}

func TestLastBolusStatusV2RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LastBolusStatusV2Request")
}
