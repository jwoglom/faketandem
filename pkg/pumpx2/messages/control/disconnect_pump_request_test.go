package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestDisconnectPumpRequest(t *testing.T) {
	parity.TestMessage(t, "01debede18da38f31f38413194c36c036e110c25", 0xde, 1, messages.Control, control.NewDisconnectPumpRequest(), "00deff81b3f10f7af28dcfb8fc")
}

func TestDisconnectPumpRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "DisconnectPumpRequest")
}
