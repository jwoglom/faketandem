package harness

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/handler"
	"github.com/jwoglom/faketandem/pkg/protocol"
	"github.com/jwoglom/faketandem/pkg/state"
)

func subscribedFromState(t *testing.T, rig *faultRig) []interface{} {
	t.Helper()
	conn, _ := mustDo(t, rig.mux, http.MethodGet, "/api/state", "")["connection"].(map[string]interface{})
	subs, ok := conn["subscribed"].([]interface{})
	if !ok {
		t.Fatalf("connection.subscribed missing or not a list: %v", conn)
	}
	return subs
}

func TestStateReportsTheCentralsSubscriptions(t *testing.T) {
	rig := newFaultRig(t)

	subs := subscribedFromState(t, rig)
	if len(subs) != 1 || subs[0] != "CURRENT_STATUS" {
		t.Fatalf("subscribed = %v, want [CURRENT_STATUS]", subs)
	}

	rig.client.Subscribe(bluetooth.QualifyingEventsCharUUID, true)
	subs = subscribedFromState(t, rig)
	if len(subs) != 2 || subs[0] != "CURRENT_STATUS" || subs[1] != "QUALIFYING_EVENTS" {
		t.Fatalf("subscribed = %v, want [CURRENT_STATUS QUALIFYING_EVENTS]", subs)
	}

	rig.client.Subscribe(bluetooth.CurrentStatusCharUUID, false)
	subs = subscribedFromState(t, rig)
	if len(subs) != 1 || subs[0] != "QUALIFYING_EVENTS" {
		t.Fatalf("subscribed = %v, want [QUALIFYING_EVENTS]", subs)
	}
}

func TestQualifyingEventSaysWhenTheCentralIsNotSubscribed(t *testing.T) {
	rig := newFaultRig(t)

	code, body := do(t, rig.mux, http.MethodPost, "/api/state/qualifyingevent", `{"bitmask":262144}`)
	if code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %v", code, body)
	}
	msg, _ := body["error"].(string)
	if !strings.Contains(msg, "not subscribed") || strings.Contains(msg, "no central attached") {
		t.Errorf("error = %q, want the not-subscribed message", msg)
	}

	rig.client.Subscribe(bluetooth.QualifyingEventsCharUUID, true)
	mustDo(t, rig.mux, http.MethodPost, "/api/state/qualifyingevent", `{"bitmask":262144}`)
}

func TestQualifyingEventSaysWhenNoCentralIsAttached(t *testing.T) {
	transport, err := bluetooth.NewVirtual(bluetooth.VirtualOptions{Addr: "127.0.0.1:0", SerialNumber: "11223344"})
	if err != nil {
		t.Fatalf("NewVirtual: %v", err)
	}
	t.Cleanup(func() { _ = transport.Close() })
	ps := state.NewPumpState()
	router := handler.NewRouter(nil, ps, transport, protocol.NewTransactionManager(time.Second),
		"go", "", "jar", "", "java", "")
	h := New(Options{PumpState: ps, Transport: transport, Router: router})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	code, body := do(t, mux, http.MethodPost, "/api/state/qualifyingevent", `{"bitmask":262144}`)
	if code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %v", code, body)
	}
	msg, _ := body["error"].(string)
	if !strings.Contains(msg, "no central attached") || strings.Contains(msg, "not subscribed") {
		t.Errorf("error = %q, want the no-central message", msg)
	}

	conn, _ := mustDo(t, mux, http.MethodGet, "/api/state", "")["connection"].(map[string]interface{})
	if subs, ok := conn["subscribed"].([]interface{}); !ok || len(subs) != 0 {
		t.Errorf("subscribed = %v, want an empty list", conn["subscribed"])
	}
}
