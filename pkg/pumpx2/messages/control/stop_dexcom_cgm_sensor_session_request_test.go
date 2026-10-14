package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStopDexcomCGMSensorSessionRequest(t *testing.T) {
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1"}}
	for _, tc := range []struct {
		name    string
		packets []string
		txID    int
	}{
		{"G6 sensor", []string{"015ab45a18810df31fafbfcb967e2fa032a42788", "005a3bfb226afd6e404f933dd7"}, 90},
		{"G7 sensor", []string{"01ecb4ec1892cf5a2085735db9a25e5ae65bf61d", "00ece4eb695f3d2116251ef498"}, -20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tester.Test(t, tc.packets[0], tc.txID, 1, messages.Control, control.NewStopDexcomCGMSensorSessionRequest(), tc.packets[1:]...)
		})
	}
}

func TestStopDexcomCGMSensorSessionRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StopDexcomCGMSensorSessionRequest")
}
