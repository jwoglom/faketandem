package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetPumpSoundsResponse(t *testing.T) {
	parity.TestMessage(t, "004ee54e1900648168208bb1895bc466198b6c049eb08f073d9afed28dba8c00", 78, 1, messages.Control, control.NewSetPumpSoundsResponse(0))
}

func TestSetPumpSoundsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetPumpSoundsResponse")
}
