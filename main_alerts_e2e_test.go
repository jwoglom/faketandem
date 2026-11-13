package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
)

// drainEvents collects the qualifying events notified within a second.
func (p *e2ePump) drainEvents() []uint32 {
	p.nextNotification("none", time.Second)
	events := p.events
	p.events = nil
	return events
}

func hasEvent(events []uint32, bit uint32) bool {
	for _, e := range events {
		if e&bit != 0 {
			return true
		}
	}
	return false
}

func (p *e2ePump) delete(path string) int {
	p.t.Helper()
	req, err := http.NewRequest(http.MethodDelete, p.apiURL+path, nil)
	if err != nil {
		p.t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		p.t.Fatal(err)
	}
	_ = resp.Body.Close() // Safe to ignore: the status is all that is read.
	return resp.StatusCode
}

// TestAlertsAndTheResumePumpAlarmOverTheLink raises notifications through the
// HTTP API and reads them as an app would: the qualifying event, the status
// read, and a ResumePumping refused until RESUME_PUMP_ALARM is dismissed.
func TestAlertsAndTheResumePumpAlarmOverTheLink(t *testing.T) {
	if testing.Short() {
		t.Skip("starts the emulator as a separate process")
	}
	p := startWithoutPumpX2(t)
	p.pair()
	p.client.Subscribe(bluetooth.QualifyingEventsCharUUID, true)
	status := bluetooth.CharCurrentStatus
	statusUUID := bluetooth.CurrentStatusCharUUID

	if code := p.post("/api/state/alert", `{"type":"LOW_INSULIN"}`); code != http.StatusOK {
		t.Fatalf("raising LOW_INSULIN answered %d", code)
	}
	if events := p.drainEvents(); !hasEvent(events, 1) {
		t.Errorf("qualifying events %v, want ALERT (1)", events)
	}
	alerts := p.exchange(status, statusUUID, "AlertStatusRequest", nil)
	if names := fmt.Sprint(alerts.Cargo["alerts"]); names != "[LOW_INSULIN_ALERT]" {
		t.Errorf("AlertStatusResponse alerts = %v, want LOW_INSULIN_ALERT", alerts.Cargo["alerts"])
	}

	p.resumeUnderTheResumePumpAlarm()

	if code := p.delete("/api/state/alert?type=LOW_INSULIN"); code != http.StatusOK {
		t.Errorf("clearing LOW_INSULIN answered %d", code)
	}
	if events := p.drainEvents(); !hasEvent(events, 1) {
		t.Errorf("qualifying events %v, want ALERT (1) for the clear", events)
	}
	alerts = p.exchange(status, statusUUID, "AlertStatusRequest", nil)
	if cargo := alerts.Cargo["cargo"]; cargo != "0000000000000000" {
		t.Errorf("AlertStatusResponse cargo = %v after the clear", cargo)
	}
}

// resumeUnderTheResumePumpAlarm stops delivery with an alarm, which raises
// RESUME_PUMP_ALARM, and resumes as TandemKit does: refused, dismissed, retried.
func (p *e2ePump) resumeUnderTheResumePumpAlarm() {
	t := p.t
	t.Helper()
	status, statusUUID := bluetooth.CharCurrentStatus, bluetooth.CurrentStatusCharUUID
	control, controlUUID := bluetooth.CharControl, bluetooth.ControlCharUUID

	if code := p.post("/api/state/suspend", `{"reason":"alarm"}`); code != http.StatusOK {
		t.Fatalf("alarm suspend answered %d", code)
	}
	if events := p.drainEvents(); !hasEvent(events, 2) || !hasEvent(events, 64) {
		t.Errorf("qualifying events %v, want ALARM (2) and PUMP_SUSPEND (64)", events)
	}
	alarms := p.exchange(status, statusUUID, "AlarmStatusRequest", nil)
	if cargo := alarms.Cargo["cargo"]; cargo != "0000040000000000" {
		t.Errorf("AlarmStatusResponse cargo = %v, want RESUME_PUMP_ALARM (bit 18)", cargo)
	}

	resume := p.exchange(control, controlUUID, "ResumePumpingRequest", nil)
	if code, _ := resume.Cargo["status"].(int); code == 0 {
		t.Fatalf("ResumePumping answered %v under RESUME_PUMP_ALARM, want a refusal", resume.Cargo)
	}

	// notificationId 18 (RESUME_PUMP_ALARM), type 2 (alarm), no extra action.
	dismiss := p.exchange(control, controlUUID, "DismissNotificationRequest", map[string]interface{}{"raw": "120000000200"})
	if code, _ := dismiss.Cargo["status"].(int); code != 0 {
		t.Fatalf("DismissNotification answered %v", dismiss.Cargo)
	}
	resume = p.exchange(control, controlUUID, "ResumePumpingRequest", nil)
	if code, _ := resume.Cargo["status"].(int); code != 0 {
		t.Errorf("ResumePumping after the dismissal answered %v", resume.Cargo)
	}
	if events := p.drainEvents(); !hasEvent(events, 128) {
		t.Errorf("qualifying events %v, want PUMP_RESUME (128)", events)
	}
	basal := p.exchange(status, statusUUID, "CurrentBasalStatusRequest", nil)
	if rate, _ := basal.Cargo["currentBasalRate"].(int); rate <= 0 {
		t.Errorf("currentBasalRate %v after the resume, want delivery", basal.Cargo["currentBasalRate"])
	}
}
