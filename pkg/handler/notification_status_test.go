package handler

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

func raise(t *testing.T, ps *state.PumpState, c state.NotificationCategory, bit uint, faultID uint32) {
	t.Helper()
	if _, ok := ps.RaiseNotification(state.Alert{Type: state.AlertNotification, Category: c, Bit: bit, FaultID: faultID}, "test"); !ok {
		t.Fatalf("%s %d not raised", c, bit)
	}
}

// readStatus runs a notification status request through its handler and parses the answer.
func readStatus(t *testing.T, bridge *pumpx2.Bridge, ps *state.PumpState, request string, cargo map[string]interface{}) *pumpx2.ParsedMessage {
	t.Helper()
	resp, err := NewNotificationStatusHandler(bridge, request).HandleMessage(&pumpx2.ParsedMessage{TxID: 3, MessageType: request, Cargo: cargo}, ps)
	if err != nil {
		t.Fatalf("%s: %v", request, err)
	}
	packets := []string(nil)
	if resp.NativeResponse != nil {
		packets = resp.NativeResponse.PacketsHex()
	} else {
		packets = resp.ResponseMessage.Packets
	}
	parsed, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, packets)
	if err != nil || !parsed.IsValid {
		t.Fatalf("%s answer does not parse: %v", request, err)
	}
	return parsed
}

func TestNotificationStatusReadsTheLiveBitmasks(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()
	raise(t, ps, state.NotificationAlert, 0, 0)            // LOW_INSULIN_ALERT
	raise(t, ps, state.NotificationMalfunction, 9, 0x2071) // MOTOR
	raise(t, ps, state.NotificationReminder, 2, 0)         // SITE_CHANGE_REMINDER
	raise(t, ps, state.NotificationCGMAlert, 3, 0)         // LOW_CGM_ALERT

	for _, tc := range []struct {
		request string
		cargo   map[string]interface{}
		want    string
	}{
		{"AlertStatusRequest", nil, "0100000000000000"},
		{"MalfunctionStatusRequest", nil, "0002000000000000"},
		{"ReminderStatusRequest", nil, "0400000000000000"},
		{"CGMAlertStatusRequest", nil, "0800000000000000"},
		{"HighestAamRequest", nil, "0900000071200000000000"},
		{"ActiveAamBitsRequest", map[string]interface{}{"cargo": "04"}, "0002000000000000000200000000000004"},
		{"ActiveAamBitsRequest", map[string]interface{}{"cargo": "02"}, "0000000000000000000000000000000002"},
	} {
		if got := readStatus(t, bridge, ps, tc.request, tc.cargo).Cargo["cargo"]; got != tc.want {
			t.Errorf("%s %v cargo = %v, want %s", tc.request, tc.cargo, got, tc.want)
		}
	}
}

func TestNotificationStatusWithNothingStanding(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()

	if got := readStatus(t, bridge, ps, "AlertStatusRequest", nil).Cargo["cargo"]; got != "0000000000000000" {
		t.Errorf("AlertStatus cargo = %v, want none", got)
	}
	if got := readStatus(t, bridge, ps, "HighestAamRequest", nil).Cargo["cargo"]; got != "0000000000000000000000" {
		t.Errorf("HighestAam cargo = %v, want 0/0", got)
	}
	if got := readStatus(t, bridge, ps, "ActiveAamBitsRequest", nil).Cargo["cargo"]; got != "0000000000000000000000000000000005" {
		t.Errorf("ActiveAamBits cargo = %v, want NONE", got)
	}
}

func TestTheSimulatorsLowReservoirAlertReachesAlertStatus(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()
	ps.SetReservoirLevel(10)
	sim := state.NewSimulator(ps, 0)
	sim.Tick()

	if got := readStatus(t, bridge, ps, "AlertStatusRequest", nil).Cargo["alerts"]; got == nil {
		t.Fatal("AlertStatus reports no alerts")
	}
	if !ps.IsNotificationStanding(state.NotificationAlert, 0) {
		t.Fatal("LOW_INSULIN_ALERT is not standing")
	}
	if got := len(alertRecords(ps, "AlertActivated")); got != 1 {
		t.Errorf("AlertActivated records = %d, want 1", got)
	}

	if _, err := NewDismissNotificationHandler(bridge).HandleMessage(dismissRequest(1, 0), ps); err != nil {
		t.Fatal(err)
	}
	sim.Tick()
	if ps.IsNotificationStanding(state.NotificationAlert, 0) {
		t.Error("the dismissed alert was raised again while the reservoir stayed low")
	}
}
