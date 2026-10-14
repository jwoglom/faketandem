package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestChangeTimeDateResponse(t *testing.T) {
	parity.TestMessage(t, "0043d743190030173920f8632cafa89dbda3879449357f9242636693d60cf4fb", 67, 1, messages.Control, control.NewChangeTimeDateResponse(0))
}

func TestChangeTimeDateResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ChangeTimeDateResponse")
}
