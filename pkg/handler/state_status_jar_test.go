package handler

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

// These tests encode each state-derived response through the real cliparser jar
// and parse it straight back, so both halves are checked: that the response can
// be encoded at all (several pumpX2 response classes cannot be, for constructor
// reasons that are invisible until you try), and that the values a driver would
// read back are the ones live pump state actually holds.
//
// Skipped unless FAKETANDEM_TEST_CLIPARSER_JAR is set.

// handleAndParse runs a handler and parses its response back into the message a
// driver would decode.
func handleAndParse(t *testing.T, bridge *pumpx2.Bridge, h MessageHandler, pumpState *state.PumpState) *pumpx2.ParsedMessage {
	t.Helper()

	// Every response exercised here is served on CURRENT_STATUS.
	const char = bluetooth.CharCurrentStatus

	req := &pumpx2.ParsedMessage{TxID: 7, MessageType: h.MessageType(), Cargo: map[string]interface{}{}}
	resp, err := h.HandleMessage(req, pumpState)
	if err != nil {
		t.Fatalf("%s handler failed: %v", h.MessageType(), err)
	}
	if resp == nil || resp.ResponseMessage == nil || len(resp.ResponseMessage.Packets) == 0 {
		t.Fatalf("%s handler produced no response packets", h.MessageType())
	}

	parsed, err := bridge.ParseMessage(char, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("failed to parse the %s response back: %v", h.MessageType(), err)
	}
	return parsed
}

// TestPumpTimestampsUsePumpEpoch checks that emitted pump timestamps are
// seconds since 2008-01-01, not Unix seconds. Sending a Unix timestamp decodes
// on the driver side as a date decades in the future, which drivers silently
// clamp -- so the bug only shows up as wrong dose times, never as an error.
func TestPumpTimestampsUsePumpEpoch(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()
	pumpState.UpdateTimeSinceReset()

	bolusStart := time.Now().Add(-30 * time.Second)
	pumpState.StartBolusWithSource(2.5, 42, state.BolusSourceBluetoothRemote, 1)
	pumpState.Bolus.StartTime = bolusStart

	parsed := handleAndParse(t, bridge, NewCurrentBolusStatusHandler(bridge), pumpState)
	if parsed.MessageType != "CurrentBolusStatusResponse" {
		t.Fatalf("parsed back as %q", parsed.MessageType)
	}

	wantTimestamp := int64(state.PumpTimeSeconds(bolusStart))
	assertCargoInt(t, parsed, wantTimestamp, "timestamp")
	assertCargoInt(t, parsed, 42, "bolusId")
	assertCargoInt(t, parsed, 2500, "requestedVolume")
	assertCargoInt(t, parsed, state.BolusSourceBluetoothRemote, "bolusSourceId")

	// Guard against the timestamp quietly reverting to a Unix value: the pump
	// epoch is 1199145600 seconds later, so a Unix timestamp is always far
	// larger than a contemporaneous pump timestamp.
	if got, _ := cargoInt(parsed, "timestamp"); got >= bolusStart.Unix() {
		t.Errorf("timestamp %d looks like a Unix timestamp, not pump-epoch seconds", got)
	}

	// Round-tripping through the epoch helpers must land back on the same second.
	if got := state.PumpTimeToWallClock(uint32(wantTimestamp)).Unix(); got != bolusStart.Unix() {
		t.Errorf("PumpTimeToWallClock round-trip = %d, want %d", got, bolusStart.Unix())
	}

	// TimeSinceResetResponse.currentTime is the same epoch.
	parsedTime := handleAndParse(t, bridge, NewTimeSinceResetHandler(bridge), pumpState)
	if parsedTime.MessageType != "TimeSinceResetResponse" {
		t.Fatalf("parsed back as %q", parsedTime.MessageType)
	}
	currentTime, ok := cargoInt(parsedTime, "currentTime")
	if !ok {
		t.Fatalf("no currentTime in %v", parsedTime.Cargo)
	}
	if currentTime >= time.Now().Unix() {
		t.Errorf("currentTime %d looks like a Unix timestamp, not pump-epoch seconds", currentTime)
	}
	if delta := currentTime - int64(state.PumpTimeSeconds(time.Now())); delta > 5 || delta < -5 {
		t.Errorf("currentTime %d is %ds away from the expected pump-epoch now", currentTime, delta)
	}
}

// TestLastBolusStatus_DerivedFromState checks that a bolus recorded by the pump
// is what LastBolusStatus reports, across all three request variants. Against
// the static all-zero constants these used to serve, a driver could never match
// the bolusId it commanded and would hold every dose unfinalized.
func TestLastBolusStatus_DerivedFromState(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()
	pumpState.UpdateTimeSinceReset()

	endTime := time.Now().Add(-2 * time.Minute)
	pumpState.RecordLastBolus(state.LastBolusRecord{
		BolusID:        42,
		RequestedUnits: 2.5,
		DeliveredUnits: 2.4,
		SourceID:       state.BolusSourceBluetoothRemote,
		TypeBitmask:    1,
		EndReasonID:    state.BolusEndReasonCompleted,
		EndTime:        endTime,
	})
	wantTimestamp := int64(state.PumpTimeSeconds(endTime))

	t.Run("V2", func(t *testing.T) {
		parsed := handleAndParse(t, bridge, NewLastBolusStatusHandler(bridge, "LastBolusStatusV2Request"), pumpState)
		if parsed.MessageType != "LastBolusStatusV2Response" {
			t.Fatalf("parsed back as %q", parsed.MessageType)
		}
		assertCargoInt(t, parsed, 42, "bolusId")
		assertCargoInt(t, parsed, 2400, "deliveredVolume")
		assertCargoInt(t, parsed, 2500, "requestedVolume")
		assertCargoInt(t, parsed, state.BolusEndReasonCompleted, "bolusStatusId")
		assertCargoInt(t, parsed, state.BolusSourceBluetoothRemote, "bolusSourceId")
		assertCargoInt(t, parsed, wantTimestamp, "timestamp")
	})

	t.Run("V3", func(t *testing.T) {
		parsed := handleAndParse(t, bridge, NewLastBolusStatusHandler(bridge, "LastBolusStatusV3Request"), pumpState)
		if parsed.MessageType != "LastBolusStatusV3Response" {
			t.Fatalf("parsed back as %q", parsed.MessageType)
		}
		// Bit 0 of the presence bitmask means "a standard bolus record is here".
		assertCargoInt(t, parsed, 1, "lastBolusTypeBitmask")
		assertCargoInt(t, parsed, 42, "standardBolusId")
		assertCargoInt(t, parsed, 2400, "standardBolusDeliveredVolume")
		assertCargoInt(t, parsed, 2500, "standardBolusRequestedVolume")
		assertCargoInt(t, parsed, wantTimestamp, "standardBolusTimestamp")
		// No extended bolus is simulated, so its section stays empty.
		assertCargoInt(t, parsed, 0, "extendedBolusId")
	})

	t.Run("V1", func(t *testing.T) {
		parsed := handleAndParse(t, bridge, NewLastBolusStatusHandler(bridge, "LastBolusStatusRequest"), pumpState)
		if parsed.MessageType != "LastBolusStatusResponse" {
			t.Fatalf("parsed back as %q", parsed.MessageType)
		}
		assertCargoInt(t, parsed, 42, "bolusId")
		assertCargoInt(t, parsed, 2400, "deliveredVolume")
		assertCargoInt(t, parsed, wantTimestamp, "timestamp")
	})
}

// TestTempRateResponse_DerivedFromState checks that a temp rate set through
// SetTempRateRequest becomes visible to the TempRateRequest query a driver uses
// to decide whether an existing temp rate must be stopped first.
func TestTempRateResponse_DerivedFromState(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()

	// Inactive to begin with.
	parsed := handleAndParse(t, bridge, NewTempRateHandler(bridge), pumpState)
	if parsed.MessageType != "TempRateResponse" {
		t.Fatalf("parsed back as %q", parsed.MessageType)
	}
	if active, ok := cargoBool(parsed, "active"); !ok || active {
		t.Errorf("expected active=false for a pump with no temp rate, got %v (ok=%v)", active, ok)
	}

	// Now set one the way SetTempRateHandler does.
	start := time.Now()
	pumpState.SetBasalState(&state.BasalState{
		CurrentRate:      1.0,
		TempBasalActive:  true,
		TempBasalRate:    1.5,
		TempBasalPercent: 150,
		TempBasalStart:   start,
		TempBasalEnd:     start.Add(60 * time.Minute),
		TempRateID:       1,
	})

	parsed = handleAndParse(t, bridge, NewTempRateHandler(bridge), pumpState)
	if active, ok := cargoBool(parsed, "active"); !ok || !active {
		t.Errorf("expected active=true after setting a temp rate, got %v (ok=%v)", active, ok)
	}
	assertCargoInt(t, parsed, 150, "percentage")
	assertCargoInt(t, parsed, int64(state.PumpTimeSeconds(start)), "startTimeRaw")
	assertCargoInt(t, parsed, 3600, "duration")
}

// TestControlIQInfo_DerivedFromState checks that closed loop is reported off by
// default. A driver that reads closedLoopEnabled=true sets "use Control-IQ" and
// then refuses to enact temp basals or manual boluses at all.
func TestControlIQInfo_DerivedFromState(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()

	for _, msgType := range []string{"ControlIQInfoV1Request", "ControlIQInfoV2Request"} {
		t.Run(msgType, func(t *testing.T) {
			parsed := handleAndParse(t, bridge, NewControlIQInfoHandler(bridge, msgType), pumpState)
			if enabled, ok := cargoBool(parsed, "closedLoopEnabled"); !ok || enabled {
				t.Errorf("closedLoopEnabled = %v (ok=%v), want false by default", enabled, ok)
			}

			pumpState.SetClosedLoopEnabled(true)
			defer pumpState.SetClosedLoopEnabled(false)

			parsed = handleAndParse(t, bridge, NewControlIQInfoHandler(bridge, msgType), pumpState)
			if enabled, ok := cargoBool(parsed, "closedLoopEnabled"); !ok || !enabled {
				t.Errorf("closedLoopEnabled = %v (ok=%v), want true after enabling it", enabled, ok)
			}
		})
	}
}

// TestCurrentBasalStatus_BitmaskMatchesDriverDecoding pins the
// basalModifiedBitmask convention: 0x01 suspend, 0x02 temp rate. Real captured
// pump traffic in pumpX2's own btsnoop logs shows bitmask 1 on a pump with
// currentBasalRate 0, TempRateResponse[active=false] and a SUSPEND home-screen
// icon, i.e. bit 0x01 means suspended.
func TestCurrentBasalStatus_BitmaskMatchesDriverDecoding(t *testing.T) {
	bridge := testBridge(t)

	handler := NewCurrentBasalStatusHandler(bridge)

	t.Run("plain basal", func(t *testing.T) {
		pumpState := state.NewPumpState()
		parsed := handleAndParse(t, bridge, handler, pumpState)
		assertCargoInt(t, parsed, 0, "basalModifiedBitmask")
	})

	t.Run("suspended", func(t *testing.T) {
		pumpState := state.NewPumpState()
		pumpState.SetPumpingSuspended(true)
		parsed := handleAndParse(t, bridge, handler, pumpState)
		assertCargoInt(t, parsed, basalModifiedSuspend, "basalModifiedBitmask")
		assertCargoInt(t, parsed, 0, "currentBasalRate")
	})

	t.Run("temp rate", func(t *testing.T) {
		pumpState := state.NewPumpState()
		pumpState.SetBasalState(&state.BasalState{
			CurrentRate:      1.0,
			TempBasalActive:  true,
			TempBasalRate:    1.5,
			TempBasalPercent: 150,
		})
		parsed := handleAndParse(t, bridge, handler, pumpState)
		assertCargoInt(t, parsed, basalModifiedTempRate, "basalModifiedBitmask")
		assertCargoInt(t, parsed, 1500, "currentBasalRate")
		assertCargoInt(t, parsed, 1000, "profileBasalRate")
	})

	t.Run("suspended during a temp rate reports suspend", func(t *testing.T) {
		// Not a true bitmask: OR-ing both bits gives 0x03, which a driver maps
		// to plain basal -- the worst of the three answers.
		pumpState := state.NewPumpState()
		pumpState.SetBasalState(&state.BasalState{
			CurrentRate:     1.0,
			TempBasalActive: true,
			TempBasalRate:   1.5,
		})
		pumpState.SetPumpingSuspended(true)
		parsed := handleAndParse(t, bridge, handler, pumpState)
		assertCargoInt(t, parsed, basalModifiedSuspend, "basalModifiedBitmask")
	})
}

// TestAPIVersionResponse_DefaultsToMobi pins the default API version to the
// Mobi 3.5 release. A driver combines this with the (Mobi) PumpVersionResponse
// fixture to decide whether the Mobi-only V3 message paths are available.
func TestAPIVersionResponse_DefaultsToMobi(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()
	parsed := handleAndParse(t, bridge, NewAPIVersionHandler(bridge), pumpState)
	if parsed.MessageType != "ApiVersionResponse" {
		t.Fatalf("parsed back as %q", parsed.MessageType)
	}
	assertCargoInt(t, parsed, 3, "majorVersion")
	assertCargoInt(t, parsed, 5, "minorVersion")
}

// TestInsulinStatusReportsWholeUnits pins the reservoir scale.
//
// InsulinStatusResponse.currentInsulinAmount is whole units. Both decoders read
// it straight through with no scaling (TandemKit's fetchReservoirStatus is
// literally "Double(response.currentInsulinAmount)"), and the real captures in
// TandemKit's PumpingSuspendedHistoryLogTests -- whose insulinAmount field is
// documented as byte-identical to this one -- carry values like 31/150/180 for
// a 200-unit cartridge.
//
// The handler used to multiply by 100, so a 137 U reservoir reached the driver
// as 13699 U: not an error anywhere, just a reservoir reading two orders of
// magnitude too large, which a host app would treat as a full cartridge that
// never empties.
func TestInsulinStatusReportsWholeUnits(t *testing.T) {
	bridge := testBridge(t)

	for _, units := range []float64{200, 137, 31, 0} {
		pumpState := state.NewPumpState()
		pumpState.SetReservoirLevel(units)

		parsed := handleAndParse(t, bridge, NewInsulinStatusHandler(bridge), pumpState)
		if parsed.MessageType != "InsulinStatusResponse" {
			t.Fatalf("parsed back as %q", parsed.MessageType)
		}
		assertCargoInt(t, parsed, int64(units), "currentInsulinAmount")
	}
}

// TestCurrentBasalStatusReportsMilliunits is the counterpart to the reservoir
// check above: the basal rates on this message ARE scaled, by 1000, which is
// what TandemKit's CurrentBasalStatusResponse divides by to get U/hr. The two
// conventions sit next to each other in the same handler file, so a test that
// pins only one of them invites "fixing" the other.
func TestCurrentBasalStatusReportsMilliunits(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()
	pumpState.SetBasalState(&state.BasalState{CurrentRate: 0.85})

	parsed := handleAndParse(t, bridge, NewCurrentBasalStatusHandler(bridge), pumpState)
	if parsed.MessageType != "CurrentBasalStatusResponse" {
		t.Fatalf("parsed back as %q", parsed.MessageType)
	}
	assertCargoInt(t, parsed, 850, "profileBasalRate")
	assertCargoInt(t, parsed, 850, "currentBasalRate")
}

// TestControlIQIOBReportsMilliunits pins the other scaled field a driver reads
// off the status messages: TandemKit documents ControlIQIOBResponse's
// pumpDisplayedIOB as milliunits.
func TestControlIQIOBReportsMilliunits(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()
	pumpState.Lock()
	pumpState.IOB = 2.5
	pumpState.Unlock()

	parsed := handleAndParse(t, bridge, NewControlIQIOBHandler(bridge, "ControlIQIOBRequest"), pumpState)
	if parsed.MessageType != "ControlIQIOBResponse" {
		t.Fatalf("parsed back as %q", parsed.MessageType)
	}
	assertCargoInt(t, parsed, 2500, "mudaliarIOB")
	assertCargoInt(t, parsed, 2500, "mudaliarTotalIOB")
}

// TestCurrentBatteryReportsPercent guards the remaining number on these
// messages that is neither units nor milliunits: battery is a plain percentage
// on both V1 and V2.
func TestCurrentBatteryReportsPercent(t *testing.T) {
	bridge := testBridge(t)

	pumpState := state.NewPumpState()
	pumpState.SetBatteryLevel(85)

	for _, version := range []string{"V1", "V2"} {
		parsed := handleAndParse(t, bridge, NewCurrentBatteryHandler(bridge, version), pumpState)
		if parsed.MessageType != "CurrentBattery"+version+"Response" {
			t.Fatalf("parsed back as %q", parsed.MessageType)
		}
		assertCargoInt(t, parsed, 85, "currentBatteryAbc")
		assertCargoInt(t, parsed, 85, "currentBatteryIbc")
	}
}
