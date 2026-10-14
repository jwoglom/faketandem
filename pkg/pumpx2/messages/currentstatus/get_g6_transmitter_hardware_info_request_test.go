package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestGetG6TransmitterHardwareInfoRequest(t *testing.T) {
	parity.TestMessage(t, "001ac41a0013da", 26, 1, messages.CurrentStatus, currentstatus.NewGetG6TransmitterHardwareInfoRequest())
}

func TestGetG6TransmitterHardwareInfoRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "GetG6TransmitterHardwareInfoRequest")
}
