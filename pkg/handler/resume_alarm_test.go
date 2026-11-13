package handler

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

func controlStatus(t *testing.T, bridge *pumpx2.Bridge, resp *Response, field string) int {
	t.Helper()
	parsed, err := bridge.ParseMessage(bluetooth.CharControl, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("parse response: %v", err)
	}
	status, ok := parsed.Cargo[field].(int)
	if !ok {
		t.Fatalf("%s has no %s: %v", parsed.MessageType, field, parsed.Cargo)
	}
	return status
}

func TestResumeIsRefusedWhileAResumePumpAlarmStandsUntilItIsDismissed(t *testing.T) {
	for _, bit := range []uint{18, 23} { // RESUME_PUMP_ALARM, RESUME_PUMP_ALARM2
		bridge := testBridge(t)
		ps := state.NewPumpState()
		ps.SuspendDelivery("alarm")
		raise(t, ps, state.NotificationAlarm, bit, 0)
		resume := roundTrip(t, bridge, bluetooth.CharControl, "ResumePumpingRequest", nil)

		resp, err := NewResumePumpingHandler(bridge).HandleMessage(resume, ps)
		if err != nil {
			t.Fatal(err)
		}
		if status := controlStatus(t, bridge, resp, "status"); status == 0 || len(resp.StateChanges) != 0 {
			t.Errorf("alarm %d: resume answered status %d with %d state changes, want a refusal", bit, status, len(resp.StateChanges))
		}

		if _, err := NewDismissNotificationHandler(bridge).HandleMessage(dismissRequest(2, int(bit)), ps); err != nil {
			t.Fatal(err)
		}
		resp, err = NewResumePumpingHandler(bridge).HandleMessage(resume, ps)
		if err != nil {
			t.Fatal(err)
		}
		if status := controlStatus(t, bridge, resp, "status"); status != 0 || len(resp.StateChanges) != 1 {
			t.Errorf("alarm %d: the retry answered status %d, want the resume", bit, status)
		}
	}
}

func TestAnOcclusionAlarmDoesNotRefuseResume(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()
	ps.SuspendDelivery("occlusion")
	raise(t, ps, state.NotificationAlarm, 2, 0)

	resp, err := NewResumePumpingHandler(bridge).HandleMessage(roundTrip(t, bridge, bluetooth.CharControl, "ResumePumpingRequest", nil), ps)
	if err != nil {
		t.Fatal(err)
	}
	if status := controlStatus(t, bridge, resp, "status"); status != 0 {
		t.Errorf("status = %d, want the resume", status)
	}
}
