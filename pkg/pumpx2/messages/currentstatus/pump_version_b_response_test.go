package currentstatus_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpVersionBResponse(t *testing.T) {
	expected, err := currentstatus.NewPumpVersionBResponse("Control-IQ+ 7.9.0.2", 0, 0, 1485742, 1004000, "0", 1011004, 251300136, "0")
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t,
		"000685063c436f6e74726f6c2d49512b20372e392e302e32000000000000000000aeab1600e0510f0030000000000000003c6d0f002889fa0e300000000000000013da",
		6, 4, messages.CurrentStatus, expected).(*currentstatus.PumpVersionBResponse)
	want := currentstatus.PumpVersionBResponse{
		SoftwareName: "Control-IQ+ 7.9.0.2", SerialNumber: 1485742, ModelNumber: 1004000, PumpRevision: "0",
		PcbPartNumberA: 1011004, PcbSerialNumberA: 251300136, PcbRevisionNumberA: "0",
	}
	want.SetCargo(parsed.Cargo())
	if !reflect.DeepEqual(*parsed, want) {
		t.Errorf("parsed %+v, want %+v", *parsed, want)
	}
}

func TestPumpVersionBResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpVersionBResponse")
}
