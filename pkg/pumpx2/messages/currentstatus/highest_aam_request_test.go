package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestHighestAamRequest(t *testing.T) {
	parity.TestMessage(t, "0007780700a224", 7, 1, messages.CurrentStatus, currentstatus.NewHighestAamRequest())
}

func TestHighestAamRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "HighestAamRequest")
}
