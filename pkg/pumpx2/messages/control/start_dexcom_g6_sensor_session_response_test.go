package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStartDexcomG6SensorSessionResponse(t *testing.T) {
	parity.TestMessage(t, "002eb32e1900cbc3f21f0a67a11d39b6d34d4936e72c6cfb468c9a4b768c6115", 46, 1, messages.Control, control.NewStartDexcomG6SensorSessionResponse(0))
}

func TestStartDexcomG6SensorSessionResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StartDexcomG6SensorSessionResponse")
}
