package handler

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/settings"
	"github.com/jwoglom/faketandem/pkg/state"
)

func TestControlIQSwitchReadsTheRawCargoWithoutAnEnabledField(t *testing.T) {
	ps := state.NewPumpState()

	applyControlIQSwitch(&pumpx2.ParsedMessage{Cargo: map[string]interface{}{"cargo": "019600012801"}}, ps)
	if !ps.IsClosedLoopEnabled() {
		t.Fatal("enabled 1 did not turn Control-IQ on")
	}
	applyControlIQSwitch(&pumpx2.ParsedMessage{Cargo: map[string]interface{}{"cargo": "009600012801"}}, ps)
	if ps.IsClosedLoopEnabled() {
		t.Error("enabled 0 did not turn Control-IQ off")
	}
}

func TestChangeControlIQSettingsSwitchesClosedLoopThroughJar(t *testing.T) {
	bridge := testBridge(t)
	ps := state.NewPumpState()
	ps.SetClock(state.NewFrozenClock(time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC)))
	sm := settings.NewManager()
	settings.RegisterDefaults(sm)
	handler := NewControlIQSettingsHandler(bridge, sm)

	for _, enabled := range []bool{true, false} {
		req := roundTrip(t, bridge, bluetooth.CharControl, "ChangeControlIQSettingsRequest", map[string]interface{}{
			"enabled":                enabled,
			"weightLbs":              150,
			"totalDailyInsulinUnits": 40,
		})
		if _, err := handler.HandleMessage(req, ps); err != nil {
			t.Fatalf("ChangeControlIQSettingsRequest: %v", err)
		}
		if got := ps.IsClosedLoopEnabled(); got != enabled {
			t.Errorf("closed loop = %v after a request with enabled %v", got, enabled)
		}
	}
}
