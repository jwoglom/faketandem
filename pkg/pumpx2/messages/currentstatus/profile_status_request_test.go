package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestProfileStatusRequest(t *testing.T) {
	parity.TestMessage(t, "00033e03006b47", 3, 1, messages.CurrentStatus, currentstatus.NewProfileStatusRequest())
}

func TestProfileStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ProfileStatusRequest")
}
