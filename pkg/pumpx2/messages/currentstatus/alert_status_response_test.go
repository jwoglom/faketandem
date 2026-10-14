package currentstatus_test

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestAlertStatusResponse(t *testing.T) {
	for _, tc := range []struct {
		name   string
		hex    string
		txID   int
		alerts []currentstatus.AlertResponseType
	}{
		{"none", "000545050800000000000000005bf2", 5, nil},
		{"device paired", "00034503080000000000004000288c", 3, []currentstatus.AlertResponseType{currentstatus.AlertResponseTypeDevicePaired}},
		{"incomplete cartridge change", "00064506080020000000000000622d", 6,
			[]currentstatus.AlertResponseType{currentstatus.AlertResponseTypeIncompleteCartridgeChangeAlert}},
		{"invalid G7 transmitter ID", "000c450c080000002000000000e1df", 12,
			[]currentstatus.AlertResponseType{currentstatus.AlertResponseTypeInvalidTransmitterID}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			intMap := big.NewInt(0)
			if len(tc.alerts) == 1 {
				intMap = tc.alerts[0].WithBit()
			}
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				currentstatus.NewAlertStatusResponse(intMap)).(*currentstatus.AlertStatusResponse)
			if len(parsed.Alerts) != len(tc.alerts) || (len(tc.alerts) > 0 && !reflect.DeepEqual(parsed.Alerts, tc.alerts)) {
				t.Errorf("alerts %v, want %v", parsed.Alerts, tc.alerts)
			}
		})
	}
}

func TestAlertStatusResponseWithCGMGraphRemoved(t *testing.T) {
	parsed, err := currentstatus.NewAlertStatusResponseFromCargo([]byte{0, 0, 0, 2, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if want := []currentstatus.AlertResponseType{currentstatus.AlertResponseTypeCGMGraphRemoved}; !reflect.DeepEqual(parsed.Alerts, want) {
		t.Errorf("alerts %v, want %v", parsed.Alerts, want)
	}
}

func TestAlertStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "AlertStatusResponse")
}
