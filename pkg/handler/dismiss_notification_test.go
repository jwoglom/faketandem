package handler

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/settings"
	"github.com/jwoglom/faketandem/pkg/state"
)

func dismissRequest(typeID, id int) *pumpx2.ParsedMessage {
	return &pumpx2.ParsedMessage{
		MessageType: "DismissNotificationRequest",
		TxID:        9,
		Cargo:       map[string]interface{}{"notificationId": id, "notificationTypeId": typeID},
	}
}

func alarmClearedRecords(ps *state.PumpState) int {
	_, last := ps.GetHistoryLogSequenceRange()
	n := 0
	for _, entry := range ps.GetHistoryLogEntries(0, last) {
		if entry.Type == "AlarmCleared" {
			n++
		}
	}
	return n
}

func TestDismissingAnAlarmTakesItOffAlarmStatus(t *testing.T) {
	ps := state.NewPumpState()
	ps.AddAlert(state.Alert{ID: 1, Type: state.AlertOcclusion, Priority: state.PriorityCritical, Message: "Occlusion detected"})
	ps.SetAlarm(5, true)
	h := NewDismissNotificationHandler(testBridge(t), nil)

	resp, err := h.HandleMessage(dismissRequest(2, 2), ps)
	if err != nil {
		t.Fatalf("DismissNotificationRequest: %v", err)
	}
	if resp.ResponseMessage == nil {
		t.Fatal("no DismissNotificationResponse")
	}

	if got, want := ps.GetAlarmBitmask(), uint64(1)<<5; got != want {
		t.Errorf("alarm bitmask = %#x, want %#x: only the dismissed alarm clears", got, want)
	}
	if got := len(ps.GetClearedAlerts()); got != 1 {
		t.Errorf("cleared alarms = %d, want 1", got)
	}
	if got := alarmClearedRecords(ps); got != 1 {
		t.Errorf("AlarmCleared records = %d, want 1", got)
	}
}

func TestDismissingAnAlarmBitSetDirectlyClearsIt(t *testing.T) {
	ps := state.NewPumpState()
	ps.SetAlarm(18, true)

	if _, err := NewDismissNotificationHandler(testBridge(t), nil).HandleMessage(dismissRequest(2, 18), ps); err != nil {
		t.Fatalf("DismissNotificationRequest: %v", err)
	}

	if got := ps.GetAlarmBitmask(); got != 0 {
		t.Errorf("alarm bitmask = %#x, want 0", got)
	}
	if got := alarmClearedRecords(ps); got != 0 {
		t.Errorf("AlarmCleared records = %d, want none for an alarm with no AlarmActivated", got)
	}
}

func TestDismissingAnAlertTakesItOffAlertStatus(t *testing.T) {
	bridge := testBridge(t)
	sm := settings.NewManager()
	settings.RegisterDefaults(sm)
	// LOW_INSULIN_ALERT (bit 2) and bit 9.
	if err := sm.UpdateConstant("AlertStatusRequest", map[string]interface{}{"raw": "0402000000000000"}); err != nil {
		t.Fatal(err)
	}

	if _, err := NewDismissNotificationHandler(bridge, sm).HandleMessage(dismissRequest(1, 2), state.NewPumpState()); err != nil {
		t.Fatalf("DismissNotificationRequest: %v", err)
	}

	config, err := sm.GetConfig("AlertStatusRequest")
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Value["raw"]; got != "0002000000000000" {
		t.Errorf("AlertStatusRequest raw = %v, want only bit 9 left", got)
	}
	resp, err := NewGenericSettingsHandler(bridge, sm, "AlertStatusRequest", true).HandleMessage(&pumpx2.ParsedMessage{TxID: 8}, nil)
	if err != nil {
		t.Fatalf("AlertStatusRequest: %v", err)
	}
	if _, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, resp.ResponseMessage.Packets); err != nil {
		t.Fatalf("parse AlertStatusResponse: %v", err)
	}
}

func TestDismissingAReminderClearsNothing(t *testing.T) {
	ps := state.NewPumpState()
	ps.SetAlarm(2, true)

	if _, err := NewDismissNotificationHandler(testBridge(t), nil).HandleMessage(dismissRequest(0, 2), ps); err != nil {
		t.Fatalf("DismissNotificationRequest: %v", err)
	}

	if got := ps.GetAlarmBitmask(); got != uint64(1)<<2 {
		t.Errorf("alarm bitmask = %#x, want the alarm left standing", got)
	}
}

func TestDismissNotificationRequestWithoutItsFieldsIsAnError(t *testing.T) {
	_, err := NewDismissNotificationHandler(testBridge(t), nil).HandleMessage(&pumpx2.ParsedMessage{TxID: 9}, state.NewPumpState())
	if err == nil {
		t.Error("a request with no notificationId was answered")
	}
}
