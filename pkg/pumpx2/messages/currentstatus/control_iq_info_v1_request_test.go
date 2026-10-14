package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestControlIQInfoV1Request(t *testing.T) {
	parity.TestMessage(t, "000368030005ab", 3, 1, messages.CurrentStatus, currentstatus.NewControlIQInfoV1Request())
}

func TestControlIQInfoV1RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ControlIQInfoV1Request")
}
