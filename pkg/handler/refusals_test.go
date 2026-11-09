package handler

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/state"
)

func setTempRate(t *testing.T, bridge *pumpx2.Bridge, ps *state.PumpState, percent int) int {
	t.Helper()
	msg := roundTrip(t, bridge, bluetooth.CharControl, "SetTempRateRequest", map[string]interface{}{"minutes": 30, "percent": percent})
	resp, err := NewSetTempRateHandler(bridge).HandleMessage(msg, ps)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.StateChanges) > 0 {
		ps.SetBasalState(resp.StateChanges[0].Data.(*state.BasalState))
	}
	return controlStatus(t, bridge, resp, "status")
}

func TestSetTempRateRefusals(t *testing.T) {
	for _, tc := range []struct {
		name    string
		stage   func(ps *state.PumpState)
		percent int
		off     func(r *state.Refusals)
	}{
		{"Control-IQ on", func(ps *state.PumpState) { ps.SetClosedLoopEnabled(true) }, 80,
			func(r *state.Refusals) { r.TempRateUnderControlIQ = false }},
		{"a temp already running", func(ps *state.PumpState) {
			ps.SetBasalState(&state.BasalState{CurrentRate: 0.85, TempBasalActive: true, TempBasalRate: 0.85, TempBasalPercent: 100, TempRateID: 1})
		}, 80, func(r *state.Refusals) { r.TempRateWhileRunning = false }},
		// A Mobi refused 17% of 0.58 U/hr.
		{"under 0.1 U/hr", func(ps *state.PumpState) { ps.SetBasalState(&state.BasalState{CurrentRate: 0.58}) }, 17,
			func(r *state.Refusals) { r.TempRateBelowMinimum = false }},
		{"above the max basal", func(ps *state.PumpState) { ps.SetMaxBasalMilliunits(1000) }, 150,
			func(r *state.Refusals) { r.TempRateAboveMaxBasal = false }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bridge := testBridge(t)
			ps := state.NewPumpState()
			tc.stage(ps)
			if status := setTempRate(t, bridge, ps, tc.percent); status == 0 {
				t.Error("the pump took it")
			}

			refusals := ps.GetRefusals()
			tc.off(&refusals)
			ps.SetRefusals(refusals)
			if status := setTempRate(t, bridge, ps, tc.percent); status != 0 {
				t.Errorf("with the refusal off, status = %d", status)
			}
		})
	}
}

func TestSetTempRateAcceptsZeroAndTheMinimum(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()
	ps.SetBasalState(&state.BasalState{CurrentRate: 0.58})

	if status := setTempRate(t, bridge, ps, 0); status != 0 {
		t.Errorf("0%% answered %d", status)
	}
	ps.SetBasalState(&state.BasalState{CurrentRate: 0.5})
	if status := setTempRate(t, bridge, ps, 20); status != 0 {
		t.Errorf("20%% of 0.5 U/hr (0.1 U/hr) answered %d", status)
	}
}

func TestBolusIsRefusedWhileBusy(t *testing.T) {
	for _, tc := range []struct {
		name  string
		stage func(ps *state.PumpState)
	}{
		{"a bolus running", func(ps *state.PumpState) { ps.StartBolus(1, ps.GetNextBolusID()) }},
		{"delivery suspended", func(ps *state.PumpState) { ps.SuspendDelivery("user") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bridge := testBridge(t)
			ps := state.NewPumpState()
			tc.stage(ps)

			permission := roundTrip(t, bridge, bluetooth.CharControl, "BolusPermissionRequest", nil)
			resp, err := NewBolusPermissionHandler(bridge).HandleMessage(permission, ps)
			if err != nil {
				t.Fatal(err)
			}
			if status := controlStatus(t, bridge, resp, "status"); status == 0 {
				t.Error("permission granted")
			}
			if nack := controlStatus(t, bridge, resp, "nackReasonId"); nack != nackInvalidPumpingState {
				t.Errorf("nackReasonId = %d", nack)
			}

			initiate := roundTrip(t, bridge, bluetooth.CharControl, "InitiateBolusRequest", map[string]interface{}{
				"totalVolume": 500, "bolusID": 9, "bolusTypeBitmask": 8,
				"foodVolume": 0, "correctionVolume": 500, "bolusCarbs": 0, "bolusBG": 0, "bolusIOB": 0,
			})
			resp, err = NewInitiateBolusHandler(bridge).HandleMessage(initiate, ps)
			if err != nil {
				t.Fatal(err)
			}
			if status := controlStatus(t, bridge, resp, "status"); status == 0 || len(resp.StateChanges) != 0 {
				t.Error("InitiateBolus started a bolus")
			}

			refusals := ps.GetRefusals()
			refusals.BolusWhileBusy = false
			ps.SetRefusals(refusals)
			resp, err = NewBolusPermissionHandler(bridge).HandleMessage(permission, ps)
			if err != nil {
				t.Fatal(err)
			}
			if status := controlStatus(t, bridge, resp, "status"); status != 0 {
				t.Error("with the refusal off, permission was refused")
			}
		})
	}
}
