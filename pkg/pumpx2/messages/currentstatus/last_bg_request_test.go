package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLastBGRequest(t *testing.T) {
	parity.TestMessage(t, "00033203000a32", 3, 1, messages.CurrentStatus, currentstatus.NewLastBGRequest())
}

func TestLastBGRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LastBGRequest")
}
