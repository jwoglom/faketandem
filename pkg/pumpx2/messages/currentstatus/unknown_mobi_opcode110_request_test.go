package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestUnknownMobiOpcode110Request(t *testing.T) {
	parity.TestMessage(t, "00e36ee3001709", -29, 1, messages.CurrentStatus, currentstatus.NewUnknownMobiOpcode110Request())
}

func TestUnknownMobiOpcode110RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "UnknownMobiOpcode110Request")
}
