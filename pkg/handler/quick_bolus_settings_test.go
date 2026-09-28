package handler

import (
	"reflect"
	"strings"
	"testing"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/pumpx2"
	"github.com/jwoglom/faketandem/pkg/settings"
)

// Cargos are the Tandem Mobi app's own writes (TandemKit#428).
func TestQuickBolusReadback(t *testing.T) {
	cases := []struct {
		name  string
		cargo string
		want  map[string]interface{}
	}{
		{"enable carrying 0.5 U", "0100f401d00701", map[string]interface{}{"quickBolusEnabledRaw": 1}},
		{"disable carrying 5 U", "00008813d00701", map[string]interface{}{"quickBolusEnabledRaw": 0}},
		{"enable at 1 U", "0100e803d00705", map[string]interface{}{"quickBolusEnabledRaw": 1, "quickBolusIncrementUnits": 1000}},
		{"increment 2 U", "0100d007d00704", map[string]interface{}{"quickBolusIncrementUnits": 2000}},
		{"switch to carbs", "01018813d00702", map[string]interface{}{"quickBolusEntryType": 1}},
		{"increment 15 g", "01018813983a08", map[string]interface{}{"quickBolusIncrementCarbs": 15000}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := quickBolusReadback(&pumpx2.ParsedMessage{Cargo: map[string]interface{}{"cargo": tc.cargo}})
			if err != nil {
				t.Fatalf("quickBolusReadback: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestQuickBolusReadback_RejectsMalformedCargo(t *testing.T) {
	for _, cargo := range []map[string]interface{}{
		{},
		{"cargo": 7},
		{"cargo": "zz"},
		{"cargo": "0100f401d007"},
	} {
		if _, err := quickBolusReadback(&pumpx2.ParsedMessage{Cargo: cargo}); err == nil {
			t.Errorf("cargo %v: expected an error", cargo)
		}
	}
}

func quickBolusGlobals(t *testing.T, sm *settings.Manager) map[string]interface{} {
	t.Helper()
	globals, err := sm.GetResponse("PumpGlobalsRequest")
	if err != nil {
		t.Fatalf("GetResponse(PumpGlobalsRequest): %v", err)
	}
	return globals
}

func TestQuickBolusSettingsHandler_LeavesUnflaggedFieldsAlone(t *testing.T) {
	sm := settings.NewManager()
	settings.RegisterDefaults(sm)
	h := NewQuickBolusSettingsHandler(nil, sm)

	// Only `enabled` is flagged: the 1 U it carries is not applied.
	h.updateReadback(&pumpx2.ParsedMessage{Cargo: map[string]interface{}{"cargo": "0000e803d00701"}})

	globals := quickBolusGlobals(t, sm)
	if globals["quickBolusEnabledRaw"] != 0 {
		t.Errorf("quickBolusEnabledRaw = %v, want 0", globals["quickBolusEnabledRaw"])
	}
	if globals["quickBolusIncrementUnits"] != 500 {
		t.Errorf("quickBolusIncrementUnits = %v, want the unchanged 500", globals["quickBolusIncrementUnits"])
	}
}

// TestQuickBolusSettings_ReadBackThroughJar is the write-then-verify TandemKit
// does: the next PumpGlobalsResponse must report what was just written.
func TestQuickBolusSettings_ReadBackThroughJar(t *testing.T) {
	bridge := testBridge(t)

	encoded, err := bridge.EncodeMessage(7, "SetQuickBolusSettingsRequest", map[string]interface{}{
		"enabled":          true,
		"modeRaw":          0,
		"incrementUnits":   2000,
		"incrementCarbs":   2000,
		"changedFieldsRaw": 5,
	})
	if err != nil && strings.Contains(err.Error(), "no constructor was found") {
		t.Skipf("cliparser predates the decoded SetQuickBolusSettingsRequest: %v", err)
	}
	if err != nil {
		t.Fatalf("encode SetQuickBolusSettingsRequest: %v", err)
	}
	req, err := bridge.ParseMessage(bluetooth.CharControl, encoded.Packets)
	if err != nil {
		t.Fatalf("parse SetQuickBolusSettingsRequest back: %v", err)
	}

	sm := settings.NewManager()
	settings.RegisterDefaults(sm)
	if err := sm.UpdateConstant("PumpGlobalsRequest", map[string]interface{}{"quickBolusEnabledRaw": 0}); err != nil {
		t.Fatalf("UpdateConstant: %v", err)
	}

	if _, err := NewQuickBolusSettingsHandler(bridge, sm).HandleMessage(req, nil); err != nil {
		t.Fatalf("SetQuickBolusSettingsRequest: %v", err)
	}
	resp, err := NewGenericSettingsHandler(bridge, sm, "PumpGlobalsRequest", true).HandleMessage(&pumpx2.ParsedMessage{TxID: 8}, nil)
	if err != nil {
		t.Fatalf("PumpGlobalsRequest: %v", err)
	}
	globals, err := bridge.ParseMessage(bluetooth.CharCurrentStatus, resp.ResponseMessage.Packets)
	if err != nil {
		t.Fatalf("parse PumpGlobalsResponse: %v", err)
	}

	assertCargoInt(t, globals, 1, "quickBolusEnabledRaw")
	assertCargoInt(t, globals, 2000, "quickBolusIncrementUnits")
	assertCargoInt(t, globals, 2000, "quickBolusIncrementCarbs")
	assertCargoInt(t, globals, 0, "quickBolusEntryType")
}
