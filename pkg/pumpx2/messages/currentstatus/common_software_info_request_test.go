package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCommonSoftwareInfoRequest(t *testing.T) {
	expected, err := currentstatus.NewCommonSoftwareInfoRequestFromCargo([]byte{0})
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "002f8e2f01006462", 47, 1, messages.CurrentStatus, expected).(*currentstatus.CommonSoftwareInfoRequest)
	if parsed.MCUType != currentstatus.MCUTypeARM {
		t.Errorf("MCU type %s, want ARM", parsed.MCUType)
	}
}

func TestCommonSoftwareInfoRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CommonSoftwareInfoRequest")
}
