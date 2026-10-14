package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetModesResponse(t *testing.T) {
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1905716"}}
	for _, tc := range []struct {
		name string
		hex  string
		txID int
	}{
		{"exercise mode on", "0079cd791900e8428b1ec8fa29dc63d8f79a1c678b339ba801061a620659e904", 121},
		{"exercise mode off", "0081cd811900fb428b1ec6956c4b467290f8ba7ee8df78dd2ca119dbef422894", -127},
		{"sleep mode on", "008acd8a190017438b1e16af1181440210dc1feed9fc36a3882124bc7bd2678e", -118},
		{"sleep mode off", "0091cd91190034438b1e5858889224eafe67bf93fc50bb2575a514602ff34e62", -111},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := control.NewSetModesResponseFromCargo([]byte{0})
			if err != nil {
				t.Fatal(err)
			}
			parsed := tester.Test(t, tc.hex, tc.txID, 2, messages.Control, expected).(*control.SetModesResponse)
			if parsed.Status != 0 {
				t.Errorf("status %d", parsed.Status)
			}
		})
	}
}

func TestSetModesResponseFromAStatusSetsOnlyThePayload(t *testing.T) {
	m := control.NewSetModesResponse(1)
	if m.Status != 0 || len(m.Cargo()) != 1 || m.Cargo()[0] != 1 {
		t.Errorf("built %+v", m)
	}
}

func TestSetModesResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetModesResponse")
}
