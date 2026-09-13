package handler

import (
	"os"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// The tests in this file round-trip real messages through the real pumpX2
// cliparser jar -- encode a request, parse the resulting BLE fragments back,
// then hand the parsed message to the handler that would serve it -- so that
// the cargo field names and value types the handlers read are checked against
// what cliparser actually emits rather than against what they were assumed to
// emit.
//
// This is the regression guard for a whole class of silent bug: a handler
// reading `msg.Cargo["units"].(float64)` when cliparser emits `totalVolume` as
// a Go int compiles, runs, matches nothing, and leaves the handler with a zero
// value. InitiateBolusRequest and SetTempRateRequest were both broken this way,
// each making its entire therapy path (bolus, temp basal) unusable.
//
// Skipped unless FAKETANDEM_TEST_CLIPARSER_JAR points at a built
// pumpx2-cliparser jar, since CI has none.

func testBridge(t *testing.T) *pumpx2.Bridge {
	t.Helper()

	jarPath := os.Getenv("FAKETANDEM_TEST_CLIPARSER_JAR")
	if jarPath == "" {
		t.Skip("FAKETANDEM_TEST_CLIPARSER_JAR not set, skipping real jar integration test")
	}

	bridge, err := pumpx2.NewBridge("", "jar", "", "java", jarPath)
	if err != nil {
		t.Fatalf("failed to create bridge: %v", err)
	}
	return bridge
}

// roundTrip encodes a message through cliparser and parses the resulting raw
// BLE fragments straight back, returning what a handler would actually receive.
func roundTrip(t *testing.T, bridge *pumpx2.Bridge, char bluetooth.CharacteristicType, name string, params map[string]interface{}) *pumpx2.ParsedMessage {
	t.Helper()

	encoded, err := bridge.EncodeMessage(7, name, params)
	if err != nil {
		t.Fatalf("failed to encode %s: %v", name, err)
	}
	if len(encoded.Packets) == 0 {
		t.Fatalf("encoding %s produced no packets", name)
	}

	parsed, err := bridge.ParseMessage(char, encoded.Packets)
	if err != nil {
		t.Fatalf("failed to parse %s back: %v", name, err)
	}
	if parsed.MessageType != name {
		t.Fatalf("round-tripped %s parsed back as %q", name, parsed.MessageType)
	}
	return parsed
}

func assertCargoInt(t *testing.T, msg *pumpx2.ParsedMessage, want int64, keys ...string) {
	t.Helper()

	got, ok := cargoInt(msg, keys...)
	if !ok {
		t.Fatalf("%s: no cargo field found under %v (cargo=%v)", msg.MessageType, keys, msg.Cargo)
	}
	if got != want {
		t.Errorf("%s: cargo %v = %d, want %d", msg.MessageType, keys, got, want)
	}
}

// TestInitiateBolusRequest_CargoFieldNames covers the bug that made every bolus
// fail: the handler read "insulin"/"units" and "bolusId", but the message
// carries "totalVolume" (in milliunits) and "bolusID".
func TestInitiateBolusRequest_CargoFieldNames(t *testing.T) {
	bridge := testBridge(t)

	msg := roundTrip(t, bridge, bluetooth.CharControl, "InitiateBolusRequest", map[string]interface{}{
		"totalVolume":      2500, // 2.5 U in milliunits
		"bolusID":          42,
		"bolusTypeBitmask": 1,
		"foodVolume":       2000,
		"correctionVolume": 500,
		"bolusCarbs":       30,
		"bolusBG":          120,
		"bolusIOB":         0,
	})

	assertCargoInt(t, msg, 2500, "totalVolume")
	assertCargoInt(t, msg, 42, "bolusID", "bolusId")

	// The old "insulin"/"units" keys must genuinely not exist, otherwise this
	// test would pass for the wrong reason.
	if _, ok := msg.Cargo["units"]; ok {
		t.Error("unexpected cargo key \"units\" -- test assumption is stale")
	}
	if _, ok := msg.Cargo["insulin"]; ok {
		t.Error("unexpected cargo key \"insulin\" -- test assumption is stale")
	}

	pumpState := state.NewPumpState()
	resp, err := NewInitiateBolusHandler(bridge).HandleMessage(msg, pumpState)
	if err != nil {
		t.Fatalf("InitiateBolusHandler rejected a real InitiateBolusRequest: %v", err)
	}
	if resp == nil || resp.ResponseMessage == nil {
		t.Fatal("InitiateBolusHandler returned no response message")
	}
	if len(resp.StateChanges) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(resp.StateChanges))
	}
	bolus, ok := resp.StateChanges[0].Data.(*state.BolusState)
	if !ok {
		t.Fatalf("expected a *state.BolusState, got %T", resp.StateChanges[0].Data)
	}
	if bolus.UnitsTotal != 2.5 {
		t.Errorf("UnitsTotal = %v, want 2.5 (2500 mU)", bolus.UnitsTotal)
	}
	if bolus.BolusID != 42 {
		t.Errorf("BolusID = %d, want 42", bolus.BolusID)
	}
	if bolus.SourceID != state.BolusSourceBluetoothRemote {
		t.Errorf("SourceID = %d, want %d (bluetoothRemoteBolus)", bolus.SourceID, state.BolusSourceBluetoothRemote)
	}
}

// TestSetTempRateRequest_CargoFieldNamesAndResponseEncodes covers both temp-rate
// blockers at once: the handler read "percentage"/"duration" where the message
// carries "percent"/"minutes", and SetTempRateResponse could not be encoded at
// all through cliparser's named constructor (pumpX2 declares size=4 but
// buildCargo emits 3 bytes), so no response was ever sent.
func TestSetTempRateRequest_CargoFieldNamesAndResponseEncodes(t *testing.T) {
	bridge := testBridge(t)

	msg := roundTrip(t, bridge, bluetooth.CharControl, "SetTempRateRequest", map[string]interface{}{
		"minutes": 60,
		"percent": 150,
	})

	assertCargoInt(t, msg, 60, "minutes")
	assertCargoInt(t, msg, 150, "percent")

	pumpState := state.NewPumpState()
	profileRate := pumpState.GetProfileBasalRate()

	resp, err := NewSetTempRateHandler(bridge).HandleMessage(msg, pumpState)
	if err != nil {
		t.Fatalf("SetTempRateHandler failed (SetTempRateResponse encode is the usual cause): %v", err)
	}
	if resp == nil || resp.ResponseMessage == nil || len(resp.ResponseMessage.Packets) == 0 {
		t.Fatal("SetTempRateHandler produced no SetTempRateResponse packets")
	}

	if len(resp.StateChanges) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(resp.StateChanges))
	}
	basal, ok := resp.StateChanges[0].Data.(*state.BasalState)
	if !ok {
		t.Fatalf("expected a *state.BasalState, got %T", resp.StateChanges[0].Data)
	}
	if !basal.TempBasalActive {
		t.Error("temp basal was not activated")
	}
	if basal.TempBasalPercent != 150 {
		t.Errorf("TempBasalPercent = %d, want 150", basal.TempBasalPercent)
	}
	wantRate := profileRate * 1.5
	if basal.TempBasalRate != wantRate {
		t.Errorf("TempBasalRate = %v, want %v (150%% of the profile rate)", basal.TempBasalRate, wantRate)
	}
	// 0 minutes would mean the simulator expires the temp rate on its next tick.
	if got := basal.TempBasalEnd.Sub(basal.TempBasalStart).Minutes(); got < 59.9 || got > 60.1 {
		t.Errorf("temp rate duration = %.2f minutes, want 60", got)
	}

	// The response must itself be parseable back into a SetTempRateResponse
	// carrying the tempRateId the handler allocated -- the whole point of the
	// raw-cargo escape hatch.
	parsedResp, err := bridge.ParseMessage(bluetooth.CharControl, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("failed to parse the SetTempRateResponse back: %v", err)
	}
	if parsedResp.MessageType != "SetTempRateResponse" {
		t.Fatalf("response parsed back as %q, want SetTempRateResponse", parsedResp.MessageType)
	}
	assertCargoInt(t, parsedResp, 0, "status")
	assertCargoInt(t, parsedResp, int64(basal.TempRateID), "tempRateId")
}

// TestHistoryLogRequest_CargoFieldNames: the message carries "startLog" and
// "numberOfLogs", not "startSequence"/"endSequence".
func TestHistoryLogRequest_CargoFieldNames(t *testing.T) {
	bridge := testBridge(t)

	msg := roundTrip(t, bridge, bluetooth.CharCurrentStatus, "HistoryLogRequest", map[string]interface{}{
		"startLog":     5,
		"numberOfLogs": 10,
	})

	assertCargoInt(t, msg, 5, "startLog")
	assertCargoInt(t, msg, 10, "numberOfLogs")
	if _, ok := msg.Cargo["startSequence"]; ok {
		t.Error("unexpected cargo key \"startSequence\" -- test assumption is stale")
	}

	pumpState := state.NewPumpState()
	if _, err := NewHistoryLogHandler(bridge).HandleMessage(msg, pumpState); err != nil {
		t.Fatalf("HistoryLogHandler rejected a real HistoryLogRequest: %v", err)
	}
}

// TestSetModesRequest_CargoFieldNames: the message carries "bitmap", not "mode",
// so the requested mode was never applied to pump state.
func TestSetModesRequest_CargoFieldNames(t *testing.T) {
	bridge := testBridge(t)

	// SetModesRequest has two single-argument constructors (int bitmap and
	// byte[] raw) and cliparser picks by arity, landing on the byte[] one; a
	// one-byte raw cargo of 0x02 is the same message.
	msg := roundTrip(t, bridge, bluetooth.CharControl, "SetModesRequest", map[string]interface{}{
		"raw": "02",
	})

	assertCargoInt(t, msg, 2, "bitmap")
	if _, ok := msg.Cargo["mode"]; ok {
		t.Error("unexpected cargo key \"mode\" -- test assumption is stale")
	}

	pumpState := state.NewPumpState()
	if _, err := NewSetModesHandler(bridge).HandleMessage(msg, pumpState); err != nil {
		t.Fatalf("SetModesHandler rejected a real SetModesRequest: %v", err)
	}
	if got := pumpState.GetControlIQMode(); got != 2 {
		t.Errorf("ControlIQMode = %d, want 2", got)
	}
}

// TestRemoteEntryAndChallengeRequests_CargoFieldNames covers the remaining
// cargo-reading handlers. Their key names were already right, but every one of
// them asserted .(float64) against values cliparser emits as Go ints, so none
// of them ever read a value either.
func TestRemoteEntryAndChallengeRequests_CargoFieldNames(t *testing.T) {
	bridge := testBridge(t)
	pumpState := state.NewPumpState()

	t.Run("RemoteBgEntryRequest", func(t *testing.T) {
		msg := roundTrip(t, bridge, bluetooth.CharControl, "RemoteBgEntryRequest", map[string]interface{}{
			"bg":                       120,
			"useForCgmCalibration":     false,
			"isAutopopBg":              false,
			"pumpTimeSecondsSinceBoot": 1000,
			"bolusId":                  42,
		})
		assertCargoInt(t, msg, 120, "bg", "bgValue")
		if _, err := NewRemoteBgEntryHandler(bridge).HandleMessage(msg, pumpState); err != nil {
			t.Fatalf("RemoteBgEntryHandler failed: %v", err)
		}
	})

	t.Run("RemoteCarbEntryRequest", func(t *testing.T) {
		msg := roundTrip(t, bridge, bluetooth.CharControl, "RemoteCarbEntryRequest", map[string]interface{}{
			"carbs":                    30,
			"pumpTimeSecondsSinceBoot": 1000,
			"bolusId":                  42,
		})
		assertCargoInt(t, msg, 30, "carbs", "carbGrams")
		if _, err := NewRemoteCarbEntryHandler(bridge).HandleMessage(msg, pumpState); err != nil {
			t.Fatalf("RemoteCarbEntryHandler failed: %v", err)
		}
	})

	t.Run("CancelBolusRequest", func(t *testing.T) {
		msg := roundTrip(t, bridge, bluetooth.CharControl, "CancelBolusRequest", map[string]interface{}{
			"bolusId": 42,
		})
		assertCargoInt(t, msg, 42, "bolusId", "bolusID")
		if _, err := NewCancelBolusHandler(bridge).HandleMessage(msg, pumpState); err != nil {
			t.Fatalf("CancelBolusHandler failed: %v", err)
		}
	})

	t.Run("CentralChallengeRequest", func(t *testing.T) {
		msg := roundTrip(t, bridge, bluetooth.CharAuthorization, "CentralChallengeRequest", map[string]interface{}{
			"appInstanceId":    123,
			"centralChallenge": "0102030405060708",
		})
		assertCargoInt(t, msg, 123, "appInstanceId", "appInstanceID")
		if _, err := NewCentralChallengeHandler(bridge).HandleMessage(msg, pumpState); err != nil {
			t.Fatalf("CentralChallengeHandler failed: %v", err)
		}
	})

	// SetSensorTypeRequest is deliberately absent: its only single-argument
	// constructors are (int cgmSensorType) and (CgmSensorType enum), cliparser
	// resolves by arity to the enum one, and it has no JSON-to-enum conversion
	// and no byte[] constructor -- so the message cannot be produced through
	// cliparser at all with any non-default value. Its handler reads
	// "cgmSensorType", which matches the field name in pumpX2's
	// SetSensorTypeRequest.java.
}
