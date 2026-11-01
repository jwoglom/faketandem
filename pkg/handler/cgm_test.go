package handler

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

func cgmCommand(t *testing.T, bridge *pumpx2.Bridge, ps *state.PumpState, name string, params map[string]interface{}) *pumpx2.ParsedMessage {
	t.Helper()
	var msg *pumpx2.ParsedMessage
	if name == "SetSensorTypeRequest" {
		// pumpX2's encoder takes only its enum constructor, which JSON cannot give.
		msg = &pumpx2.ParsedMessage{MessageType: name, TxID: 7, Cargo: params}
	} else {
		msg = roundTrip(t, bridge, bluetooth.CharControl, name, params)
	}
	resp, err := NewCGMCommandHandler(bridge, name).HandleMessage(msg, ps)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	parsed, err := bridge.ParseMessage(bluetooth.CharControl, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("parsing the answer to %s: %v", name, err)
	}
	return parsed
}

func cgmRead(t *testing.T, bridge *pumpx2.Bridge, ps *state.PumpState, name string) *pumpx2.ParsedMessage {
	t.Helper()
	resp, err := NewCGMStatusHandler(bridge, name).HandleMessage(roundTrip(t, bridge, bluetooth.CharCurrentStatus, name, nil), ps)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	parsed, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("parsing the answer to %s: %v", name, err)
	}
	return parsed
}

// assertCGMAlerts reads CGMAlertStatusResponse's mask from its cargo, which is
// all pumpX2 prints of it: eight bytes, little-endian.
func assertCGMAlerts(t *testing.T, msg *pumpx2.ParsedMessage, want uint64) {
	t.Helper()
	raw, ok := cargoValue(msg, "cargo")
	b, err := hex.DecodeString(fmt.Sprint(raw))
	if !ok || err != nil || len(b) != 8 {
		t.Fatalf("CGMAlertStatusResponse cargo = %v", msg.Cargo)
	}
	if got := binary.LittleEndian.Uint64(b); got != want {
		t.Errorf("CGM alert mask = %#x, want %#x", got, want)
	}
}

func cgmPumpState(t *testing.T, sensorType int) (*state.PumpState, *state.ManualClock) {
	t.Helper()
	ps := state.NewPumpState()
	clock := state.NewFrozenClock(time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC))
	ps.SetClock(clock)
	ps.ResetCGM(sensorType)
	return ps, clock
}

func TestAG7PairingIsReadBackThroughTheStatusMessages(t *testing.T) {
	bridge := testBridge(t)
	ps, clock := cgmPumpState(t, state.CGMSensorTypeNone)

	assertCargoInt(t, cgmCommand(t, bridge, ps, "SetSensorTypeRequest", map[string]interface{}{"cgmSensorType": 3}), 0, "status")
	assertCargoInt(t, cgmCommand(t, bridge, ps, "SetDexcomG7PairingCodeRequest", map[string]interface{}{"pairingCode": 1234}), 0, "status")

	v2 := cgmRead(t, bridge, ps, "CgmStatusV2Request")
	assertCargoInt(t, v2, state.CGMSessionStartPending, "sessionStateId")
	assertCargoInt(t, v2, state.CGMSensorTypeDexcomG7, "cgmSensorTypeId")
	assertCargoInt(t, cgmRead(t, bridge, ps, "GetSavedG7PairingCodeRequest"), 1234, "pairingCode")

	clock.Advance(30*time.Second + 27*time.Minute + 6*time.Minute)
	v2 = cgmRead(t, bridge, ps, "CgmStatusV2Request")
	assertCargoInt(t, v2, state.CGMSessionActive, "sessionStateId")
	assertCargoInt(t, v2, int64(10*24*time.Hour/time.Second), "sessionDurationSeconds")
	assertCargoInt(t, v2, int64((10*24*time.Hour-33*time.Minute)/time.Second), "sessionTimeRemainingSeconds")
	assertCargoInt(t, v2, int64(ps.PumpTimeFor(time.Date(2024, time.March, 5, 12, 0, 30, 0, time.UTC))), "sensorStartedTimestamp")
	assertCargoInt(t, v2, state.CGMBatteryUnavailable, "transmitterBatteryStatusId")

	egv := cgmRead(t, bridge, ps, "CurrentEGVGuiDataRequest")
	assertCargoInt(t, egv, 120, "cgmReading")
	assertCargoInt(t, egv, state.EGVStatusValid, "egvStatusId")
	assertCargoInt(t, egv, int64(ps.PumpTimeFor(time.Date(2024, time.March, 5, 12, 32, 30, 0, time.UTC))), "bgReadingTimestampSeconds")
	assertCargoInt(t, cgmRead(t, bridge, ps, "CurrentEgvGuiDataV2Request"), 120, "cgmReading")
	assertCargoInt(t, cgmRead(t, bridge, ps, "CGMStatusRequest"), state.CGMSessionActive, "sessionStateId")
	assertCGMAlerts(t, cgmRead(t, bridge, ps, "CGMAlertStatusRequest"), 0)

	assertCargoInt(t, cgmCommand(t, bridge, ps, "StopDexcomCGMSensorSessionRequest", nil), 0, "status")
	assertCargoInt(t, cgmRead(t, bridge, ps, "CgmStatusV2Request"), state.CGMSessionStopPending, "sessionStateId")
	clock.Advance(5 * time.Second)
	v2 = cgmRead(t, bridge, ps, "CgmStatusV2Request")
	assertCargoInt(t, v2, state.CGMSessionStopped, "sessionStateId")
	assertCargoInt(t, v2, 0, "sensorStartedTimestamp")
	assertCargoInt(t, cgmRead(t, bridge, ps, "CurrentEGVGuiDataRequest"), state.EGVStatusUnavailable, "egvStatusId")
}

func TestAG6TransmitterIsReadBackAsTheHardwareInfo(t *testing.T) {
	bridge := testBridge(t)
	ps, _ := cgmPumpState(t, state.CGMSensorTypeDexcomG6)

	assertCargoInt(t, cgmCommand(t, bridge, ps, "SetG6TransmitterIdRequest", map[string]interface{}{"txId": "8GXYZ1"}), 0, "status")
	assertCargoInt(t, cgmCommand(t, bridge, ps, "StartDexcomG6SensorSessionRequest", map[string]interface{}{"sensorCode": 5678}), 0, "status")

	info := cgmRead(t, bridge, ps, "CGMHardwareInfoRequest")
	if got, _ := cargoValue(info, "hardwareInfoString"); got != "8GXYZ1" {
		t.Errorf("hardwareInfoString = %v", got)
	}
	v2 := cgmRead(t, bridge, ps, "CgmStatusV2Request")
	assertCargoInt(t, v2, state.CGMSessionStartPending, "sessionStateId")
	assertCargoInt(t, v2, state.CGMBatteryOK, "transmitterBatteryStatusId")
}

func TestARefusedCGMCommandAnswersANonzeroStatus(t *testing.T) {
	bridge := testBridge(t)
	ps, _ := cgmPumpState(t, state.CGMSensorTypeDexcomG6)

	assertCargoInt(t, cgmCommand(t, bridge, ps, "SetDexcomG7PairingCodeRequest", map[string]interface{}{"pairingCode": 1234}), 1, "status")

	_ = ps.UpdateCGM(func(c *state.CGMState, _ time.Time) error {
		c.Refuse = map[string]int{"SetSensorTypeRequest": 7}
		return nil
	})
	assertCargoInt(t, cgmCommand(t, bridge, ps, "SetSensorTypeRequest", map[string]interface{}{"cgmSensorType": 3}), 7, "status")
	if ps.CGM().SensorType != state.CGMSensorTypeDexcomG6 {
		t.Error("a refused SetSensorType changed the sensor type")
	}
}

func TestDismissingACGMAlertTakesItOffCGMAlertStatus(t *testing.T) {
	bridge := testBridge(t)
	ps, _ := cgmPumpState(t, state.CGMSensorTypeDexcomG7)
	_ = ps.UpdateCGM(func(c *state.CGMState, now time.Time) error { return c.MoveTo(state.CGMEndExpired, now, nil) })
	assertCGMAlerts(t, cgmRead(t, bridge, ps, "CGMAlertStatusRequest"), 1<<state.CGMAlertSensorExpired)

	if _, err := NewDismissNotificationHandler(bridge, nil).HandleMessage(dismissRequest(3, state.CGMAlertSensorExpired), ps); err != nil {
		t.Fatalf("DismissNotificationRequest: %v", err)
	}
	assertCGMAlerts(t, cgmRead(t, bridge, ps, "CGMAlertStatusRequest"), 0)
}
