package harness

import (
	"net/http"
	"testing"

	"github.com/jwoglom/faketandem/pkg/state"
)

func TestRaisingAnAlertByNameSetsItsBitAndWritesItsRecord(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	body := mustDo(t, mux, http.MethodPost, "/api/state/alert", `{"type":"LOW_INSULIN"}`)

	raised := body["raised"].(map[string]interface{})
	if raised["name"] != "LOW_INSULIN_ALERT" || raised["category"] != "alert" || raised["bit"] != float64(0) {
		t.Errorf("raised = %v", raised)
	}
	if got := ps.NotificationBitmask(state.NotificationAlert); got != 1 {
		t.Errorf("alert bitmask = %#x, want LOW_INSULIN_ALERT", got)
	}
	activated := recordsOfType(historyEntries(t, mux), "AlertActivated")
	if len(activated) != 1 || fieldsOf(activated[0], "data")["alertId"] != float64(0) {
		t.Errorf("AlertActivated records = %v", activated)
	}
	assertHasBit(t, transport.qualifyingEventBits(), 1, "alert")

	notifications := body["state"].(map[string]interface{})["notifications"].(map[string]interface{})
	alert := notifications["alert"].(map[string]interface{})
	if alert["bitmask"] != "1" || alert["names"].([]interface{})[0] != "LOW_INSULIN_ALERT" {
		t.Errorf("snapshot notifications.alert = %v", alert)
	}

	if code, _ := do(t, mux, http.MethodPost, "/api/state/alert", `{"type":"LOW_INSULIN_ALERT"}`); code != http.StatusConflict {
		t.Errorf("raising a standing alert again = %d, want 409", code)
	}

	mustDo(t, mux, http.MethodDelete, "/api/state/alert", `{"type":"LOW_INSULIN"}`)
	if got := ps.NotificationBitmask(state.NotificationAlert); got != 0 {
		t.Errorf("alert bitmask = %#x after DELETE", got)
	}
	if got := len(recordsOfType(historyEntries(t, mux), "AlertCleared")); got != 1 {
		t.Errorf("AlertCleared records = %d, want 1", got)
	}
	if code, _ := do(t, mux, http.MethodDelete, "/api/state/alert?type=LOW_INSULIN", ""); code != http.StatusConflict {
		t.Errorf("clearing an alert that does not stand = %d, want 409", code)
	}
}

func TestRaisingAnAlarmRaisesTheAlarmEvent(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/alarm", `{"type":"RESUME_PUMP_ALARM"}`)

	if !ps.IsNotificationStanding(state.NotificationAlarm, 18) {
		t.Error("RESUME_PUMP_ALARM does not stand")
	}
	assertHasBit(t, transport.qualifyingEventBits(), 2, "alarm")
	activated := recordsOfType(historyEntries(t, mux), "AlarmActivated")
	if len(activated) != 1 || fieldsOf(activated[0], "data")["alarmId"] != float64(18) {
		t.Errorf("AlarmActivated records = %v", activated)
	}

	mustDo(t, mux, http.MethodDelete, "/api/state/alarm", `{"code":18}`)
	if got := len(recordsOfType(historyEntries(t, mux), "AlarmCleared")); got != 1 {
		t.Errorf("AlarmCleared records = %d, want 1", got)
	}
}

func TestRaisingAMalfunction(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/malfunction", `{"code":9,"fault_id":4660}`)

	highest, ok := ps.HighestNotification()
	if !ok || highest.Category != state.NotificationMalfunction || highest.Bit != 9 || highest.FaultID != 4660 {
		t.Errorf("highest = %+v (%v), want malfunction 9 with fault 4660", highest, ok)
	}
	assertHasBit(t, transport.qualifyingEventBits(), 8, "malfunction")
	if got := len(recordsOfType(historyEntries(t, mux), "Malfunction")); got != 1 {
		t.Errorf("Malfunction records = %d, want 1", got)
	}

	// A resume acknowledges alarms, never a malfunction; clear_alerts staging does.
	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"alarm"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/resume", "")
	if !ps.IsNotificationStanding(state.NotificationMalfunction, 9) {
		t.Error("a resume cleared the malfunction")
	}
	mustDo(t, mux, http.MethodPut, "/api/state", `{"clear_alerts":true}`)
	if ps.IsNotificationStanding(state.NotificationMalfunction, 9) {
		t.Error("clear_alerts left the malfunction")
	}
}

func TestReminderAndCGMAlert(t *testing.T) {
	_, ps, transport, mux := testHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/reminder", `{"type":"SITE_CHANGE"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/cgmalert", `{"type":"LOW"}`)

	if got := ps.NotificationBitmask(state.NotificationReminder); got != 1<<2 {
		t.Errorf("reminder bitmask = %#x, want SITE_CHANGE_REMINDER", got)
	}
	if got := ps.NotificationBitmask(state.NotificationCGMAlert); got != 1<<3 {
		t.Errorf("CGM alert bitmask = %#x, want LOW_CGM_ALERT", got)
	}
	bits := transport.qualifyingEventBits()
	assertHasBit(t, bits, 4, "reminder")
	assertHasBit(t, bits, 16, "cgmAlert")
}

func TestUnknownNotificationNamesAreRejected(t *testing.T) {
	_, _, _, mux := testHarness(t)

	for _, body := range []string{`{"type":"NOT_AN_ALERT"}`, `{}`, `{"code":64}`} {
		if code, _ := do(t, mux, http.MethodPost, "/api/state/alert", body); code != http.StatusBadRequest {
			t.Errorf("POST alert %s = %d, want 400", body, code)
		}
	}
	if code, _ := do(t, mux, http.MethodDelete, "/api/state/bolus/start", ""); code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE on an action with no clear = %d, want 405", code)
	}
}

func TestAnAlarmSuspendRaisesResumePumpAlarm(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"alarm"}`)

	if !ps.IsNotificationStanding(state.NotificationAlarm, 18) {
		t.Error("no RESUME_PUMP_ALARM after an alarm suspend")
	}
	mustDo(t, mux, http.MethodPost, "/api/state/resume", `{"clear_alarms":false}`)
	if !ps.IsNotificationStanding(state.NotificationAlarm, 18) {
		t.Error("clear_alarms false cleared RESUME_PUMP_ALARM")
	}
}
