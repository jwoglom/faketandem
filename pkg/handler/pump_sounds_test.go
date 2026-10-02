package handler

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/settings"
)

func TestPumpSoundsReadbackAppliesOnlyFlaggedSounds(t *testing.T) {
	// Quick bolus 2, general 3, reminder 1, alert 0, alarm 2; general and alarm flagged.
	got, err := pumpSoundsReadback(&pumpx2.ParsedMessage{Cargo: map[string]interface{}{"cargo": "000203010002000024"}})
	if err != nil {
		t.Fatalf("pumpSoundsReadback: %v", err)
	}
	if want := map[string]interface{}{"bolusAnnun": 3, "alarmAnnun": 2}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := pumpSoundsReadback(&pumpx2.ParsedMessage{Cargo: map[string]interface{}{"cargo": "0002"}}); err == nil {
		t.Error("a short cargo was accepted")
	}
}

func TestPumpSounds_ReadBackThroughJar(t *testing.T) {
	bridge := testBridge(t)
	req := roundTrip(t, bridge, bluetooth.CharControl, "SetPumpSoundsRequest", map[string]interface{}{
		"quickBolusAnnunRaw": 1,
		"generalAnnunRaw":    2,
		"reminderAnnunRaw":   3,
		"alertAnnunRaw":      0,
		"alarmAnnunRaw":      1,
		"cgmAlertAnnunA":     0,
		"cgmAlertAnnunB":     0,
		"changeBitmaskRaw":   0x3e,
	})

	sm := settings.NewManager()
	settings.RegisterDefaults(sm)
	if _, err := NewPumpSoundsHandler(bridge, sm).HandleMessage(req, nil); err != nil {
		t.Fatalf("SetPumpSoundsRequest: %v", err)
	}
	resp, err := NewGenericSettingsHandler(bridge, sm, "PumpGlobalsRequest", true).HandleMessage(&pumpx2.ParsedMessage{TxID: 8}, nil)
	if err != nil {
		t.Fatalf("PumpGlobalsRequest: %v", err)
	}
	globals, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("parse PumpGlobalsResponse: %v", err)
	}
	assertCargoInt(t, globals, 1, "quickBolusAnnun")
	assertCargoInt(t, globals, 2, "bolusAnnun", "generalAnnun")
	assertCargoInt(t, globals, 3, "reminderAnnun")
	assertCargoInt(t, globals, 0, "alertAnnun")
	assertCargoInt(t, globals, 1, "alarmAnnun")
}
