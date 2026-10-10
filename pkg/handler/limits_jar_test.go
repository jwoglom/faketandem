package handler

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

func TestDeliveryLimitWritesAreReadBack(t *testing.T) {
	bridge := testBridge(t)
	pumpState := state.NewPumpState()

	bolus := roundTrip(t, bridge, bluetooth.CharControl, "SetMaxBolusLimitRequest", map[string]interface{}{"maxBolusMilliunits": 8000})
	assertCargoInt(t, bolus, 8000, "maxBolusMilliunits")
	if _, err := NewDeliveryLimitWriteHandler(bridge, "SetMaxBolusLimitRequest").HandleMessage(bolus, pumpState); err != nil {
		t.Fatalf("SetMaxBolusLimitRequest: %v", err)
	}
	basal := roundTrip(t, bridge, bluetooth.CharControl, "SetMaxBasalLimitRequest", map[string]interface{}{"maxHourlyBasalMilliunits": 2500})
	assertCargoInt(t, basal, 2500, "maxHourlyBasalMilliunits")
	if _, err := NewDeliveryLimitWriteHandler(bridge, "SetMaxBasalLimitRequest").HandleMessage(basal, pumpState); err != nil {
		t.Fatalf("SetMaxBasalLimitRequest: %v", err)
	}

	limits := pumpState.GetDeliveryLimits()
	if limits.MaxBolusMilliunits != 8000 || limits.MaxBasalMilliunits != 2500 {
		t.Fatalf("limits after the writes = %+v", limits)
	}

	for name, field := range map[string]string{"GlobalMaxBolusSettingsRequest": "maxBolus", "BasalLimitSettingsRequest": "basalLimit"} {
		resp, err := NewDeliveryLimitsReadHandler(bridge, name).HandleMessage(roundTrip(t, bridge, bluetooth.CharCurrentStatus, name, nil), pumpState)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		parsed, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, resp.ResponseMessage.Packets)
		if err != nil {
			t.Fatalf("parsing the answer to %s: %v", name, err)
		}
		want := map[string]int64{"maxBolus": 8000, "basalLimit": 2500}[field]
		assertCargoInt(t, parsed, want, field)
	}
}

func TestInitiateBolusAboveTheMaxBolusIsRefused(t *testing.T) {
	bridge := testBridge(t)
	pumpState := state.NewPumpState()
	pumpState.SetMaxBolusMilliunits(2000)

	msg := roundTrip(t, bridge, bluetooth.CharControl, "InitiateBolusRequest", map[string]interface{}{
		"totalVolume": 2500, "bolusID": 43, "bolusTypeBitmask": 1, "foodVolume": 2500,
		"correctionVolume": 0, "bolusCarbs": 0, "bolusBG": 0, "bolusIOB": 0,
	})
	resp, err := NewInitiateBolusHandler(bridge).HandleMessage(msg, pumpState)
	if err != nil {
		t.Fatalf("InitiateBolusHandler: %v", err)
	}
	if len(resp.StateChanges) != 0 {
		t.Errorf("a refused bolus changed state: %+v", resp.StateChanges)
	}
	parsed, err := bridge.ParseMessage(bluetooth.CharControl, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("parsing InitiateBolusResponse: %v", err)
	}
	assertCargoInt(t, parsed, 1, "status")
}

// pumpX2's encoder will not build a SetTempRateRequest outside its bounds, so the request is built by hand.
func TestSetTempRateOutsideThePumpsBoundsIsRefused(t *testing.T) {
	bridge := testBridge(t)

	for _, c := range []struct {
		percent, minutes int
		accepted         bool
	}{
		{250, 30, true}, {0, 15, true}, {100, 72 * 60, true},
		{251, 30, false}, {300, 30, false}, {-1, 30, false}, {150, 14, false}, {150, 72*60 + 1, false},
	} {
		msg := &pumpx2.ParsedMessage{
			MessageType: "SetTempRateRequest",
			TxID:        7,
			Cargo:       map[string]interface{}{"percent": c.percent, "minutes": c.minutes},
		}
		resp, err := NewSetTempRateHandler(bridge).HandleMessage(msg, state.NewPumpState())
		if err != nil {
			t.Fatalf("%d%% for %d min: %v", c.percent, c.minutes, err)
		}
		if started := len(resp.StateChanges) == 1; started != c.accepted {
			t.Errorf("%d%% for %d min: started a temp rate = %v, want %v", c.percent, c.minutes, started, c.accepted)
		}
		parsed, err := bridge.ParseMessage(bluetooth.CharControl, resp.ResponseMessage.Packets)
		if err != nil {
			t.Fatalf("parsing SetTempRateResponse: %v", err)
		}
		wantStatus := int64(1)
		if c.accepted {
			wantStatus = 0
		}
		assertCargoInt(t, parsed, wantStatus, "status")
	}
}
