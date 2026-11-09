package handler

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
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
	h := NewDismissNotificationHandler(testBridge(t))

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

	if _, err := NewDismissNotificationHandler(testBridge(t)).HandleMessage(dismissRequest(2, 18), ps); err != nil {
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
	ps := state.NewPumpState()
	for _, bit := range []uint{0, 9} {
		if _, ok := ps.RaiseNotification(state.Alert{Type: state.AlertNotification, Category: state.NotificationAlert, Bit: bit}, "test"); !ok {
			t.Fatalf("alert %d not raised", bit)
		}
	}

	resp, err := NewDismissNotificationHandler(bridge).HandleMessage(dismissRequest(1, 0), ps)
	if err != nil {
		t.Fatalf("DismissNotificationRequest: %v", err)
	}
	if status := dismissStatus(t, bridge, resp); status != 0 {
		t.Errorf("status = %d, want 0", status)
	}

	if got := ps.NotificationBitmask(state.NotificationAlert); got != 1<<9 {
		t.Errorf("alert bitmask = %#x, want only bit 9 left", got)
	}
	resp, err = NewNotificationStatusHandler(bridge, "AlertStatusRequest").HandleMessage(&pumpx2.ParsedMessage{TxID: 8}, ps)
	if err != nil {
		t.Fatalf("AlertStatusRequest: %v", err)
	}
	if _, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, resp.ResponseMessage.Packets); err != nil {
		t.Fatalf("parse AlertStatusResponse: %v", err)
	}
	if got := len(alertRecords(ps, "AlertCleared")); got != 1 {
		t.Errorf("AlertCleared records = %d, want 1", got)
	}
}

func TestAMalfunctionCannotBeDismissed(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()
	ps.RaiseNotification(state.Alert{Type: state.AlertNotification, Category: state.NotificationMalfunction, Bit: 9}, "test")

	resp, err := NewDismissNotificationHandler(bridge).HandleMessage(dismissRequest(4, 9), ps)
	if err != nil {
		t.Fatalf("DismissNotificationRequest: %v", err)
	}
	if status := dismissStatus(t, bridge, resp); status == 0 {
		t.Error("a malfunction's dismissal answered status 0")
	}
	if !ps.IsNotificationStanding(state.NotificationMalfunction, 9) {
		t.Error("the malfunction was dismissed")
	}
}

func dismissStatus(t *testing.T, bridge *pumpx2.Bridge, resp *Response) int {
	t.Helper()
	parsed, err := bridge.ParseMessage(bluetooth.CharControl, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("parse DismissNotificationResponse: %v", err)
	}
	status, _ := parsed.Cargo["status"].(int)
	return status
}

func alertRecords(ps *state.PumpState, typeName string) []state.HistoryLogEntry {
	_, last := ps.GetHistoryLogSequenceRange()
	var out []state.HistoryLogEntry
	for _, entry := range ps.GetHistoryLogEntries(0, last) {
		if entry.Type == typeName {
			out = append(out, entry)
		}
	}
	return out
}

func TestDismissingAReminderClearsNothing(t *testing.T) {
	ps := state.NewPumpState()
	ps.SetAlarm(2, true)

	if _, err := NewDismissNotificationHandler(testBridge(t)).HandleMessage(dismissRequest(0, 2), ps); err != nil {
		t.Fatalf("DismissNotificationRequest: %v", err)
	}

	if got := ps.GetAlarmBitmask(); got != uint64(1)<<2 {
		t.Errorf("alarm bitmask = %#x, want the alarm left standing", got)
	}
}

func TestDismissNotificationRequestWithoutItsFieldsIsAnError(t *testing.T) {
	_, err := NewDismissNotificationHandler(testBridge(t)).HandleMessage(&pumpx2.ParsedMessage{TxID: 9}, state.NewPumpState())
	if err == nil {
		t.Error("a request with no notificationId was answered")
	}
}
