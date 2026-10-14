package control_test

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// quickBolusFields are a SetQuickBolusSettingsRequest's fields as pumpX2's
// tests assert them.
type quickBolusFields struct {
	enabled                        bool
	mode                           control.QuickBolusMode
	incrementUnits, incrementCarbs int32
	changed                        []control.ChangedField
}

func (f quickBolusFields) check(t *testing.T, m *control.SetQuickBolusSettingsRequest) {
	t.Helper()
	if m.Enabled != f.enabled || m.Mode() == nil || *m.Mode() != f.mode || m.IncrementUnits != f.incrementUnits || m.IncrementCarbs != f.incrementCarbs ||
		!reflect.DeepEqual(m.ChangedFields(), f.changed) {
		t.Errorf("parsed %+v (mode %v, changed %v), want %+v", m, m.Mode(), m.ChangedFields(), f)
	}
}

func quickBolusRequestFromHex(t *testing.T, s string) *control.SetQuickBolusSettingsRequest {
	t.Helper()
	raw, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	m, err := control.NewSetQuickBolusSettingsRequestFromCargo(raw)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// pumpGlobals is a PumpGlobalsResponse with the given quick bolus settings.
func pumpGlobals(enabled bool, mode control.QuickBolusMode, incrementUnits, incrementCarbs int32) *currentstatus.PumpGlobalsResponse {
	var enabledRaw int32
	if enabled {
		enabledRaw = 1
	}
	return currentstatus.NewPumpGlobalsResponse(enabledRaw, incrementUnits, incrementCarbs, mode.Raw(), 1, 0, 0, 0, 0, 0, 0, 0)
}

// The Tandem Mobi app's own writes, each carrying the pump's current value in
// every field it does not change.
func TestSetQuickBolusSettingsRequest(t *testing.T) {
	units, carbs := control.QuickBolusModeUnits, control.QuickBolusModeCarbs
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1"}}
	for _, tc := range []struct {
		name      string
		cargo     string
		packets   []string
		txID      int
		want      quickBolusFields
		increment control.QuickBolusIncrement
	}{
		{"disable carrying half a unit", "0000f401d00701", []string{"0127d2271f0000f401d0070192b3f01f4553dc52", "0027653feef6a3f69bbed8a924ba965729772f31"}, 39,
			quickBolusFields{false, units, 500, 2000, []control.ChangedField{control.ChangedFieldEnabled}}, control.QuickBolusIncrementDisabled},
		{"enable carrying half a unit", "0100f401d00701", []string{"012fd22f1f0100f401d0070199b3f01f7b890678", "002fc35d3e9c65a3b5bde600a7a1aab82334acba"}, 47,
			quickBolusFields{true, units, 500, 2000, []control.ChangedField{control.ChangedFieldEnabled}}, control.QuickBolusIncrementUnits0_5},
		{"increment one unit", "0100e803d00704", []string{"0143d2431f0100e803d0070444b5f01ff65606f6", "0043a06ada7fe304173af8dec2e3741e2fa6b07a"}, 67,
			quickBolusFields{true, units, 1000, 2000, []control.ChangedField{control.ChangedFieldIncrementUnits}}, control.QuickBolusIncrementUnits1_0},
		{"increment two units", "0100d007d00704", []string{"0145d2451f0100d007d007045fb5f01fdf82adc2", "00458f3f2a95494ed7fe9486d8c010f179e7f356"}, 69,
			quickBolusFields{true, units, 2000, 2000, []control.ChangedField{control.ChangedFieldIncrementUnits}}, control.QuickBolusIncrementUnits2_0},
		{"increment five units", "01008813d00704", []string{"014ed24e1f01008813d007047db5f01f0925f3b8", "004e7e7ff27351e18c8547c8a0f9aa1f5c7d39ad"}, 78,
			quickBolusFields{true, units, 5000, 2000, []control.ChangedField{control.ChangedFieldIncrementUnits}}, control.QuickBolusIncrementUnits5_0},
		{"increment two grams", "01018813d00708", []string{"01b2d2b21f01018813d0070851b7f01ff804b107", "00b2a7cb9e77051542d5c9d8b5253269836e201a"}, -78,
			quickBolusFields{true, carbs, 5000, 2000, []control.ChangedField{control.ChangedFieldIncrementCarbs}}, control.QuickBolusIncrementCarbs2G},
		{"increment five grams", "01018813881308", []string{"0159d2591f01018813881308bdb5f01f9cd3f104", "005905fe4933df7b80cada7ae5e4f8f907056791"}, 89,
			quickBolusFields{true, carbs, 5000, 5000, []control.ChangedField{control.ChangedFieldIncrementCarbs}}, control.QuickBolusIncrementCarbs5G},
		{"increment ten grams", "01018813102708", []string{"015bd25b1f01018813102708d8b5f01f6f9db561", "005b23cd6e7900a0ebf7c9d4d626af1d33b5aa93"}, 91,
			quickBolusFields{true, carbs, 5000, 10000, []control.ChangedField{control.ChangedFieldIncrementCarbs}}, control.QuickBolusIncrementCarbs10G},
		{"increment fifteen grams", "01018813983a08", []string{"0164d2641f01018813983a08efb5f01f09fc0867", "006415ef922302f56194ccaf44fffd72e3276df0"}, 100,
			quickBolusFields{true, carbs, 5000, 15000, []control.ChangedField{control.ChangedFieldIncrementCarbs}}, control.QuickBolusIncrementCarbs15G},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := quickBolusRequestFromHex(t, tc.cargo)
			parsed := tester.Test(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...).(*control.SetQuickBolusSettingsRequest)
			tc.want.check(t, parsed)
			if got := parsed.Increment(); got == nil || *got != tc.increment {
				t.Errorf("increment %v, want %v", got, tc.increment)
			}
		})
	}
}

func TestSetQuickBolusSettingsRequestCargo(t *testing.T) {
	units, carbs := control.QuickBolusModeUnits, control.QuickBolusModeCarbs
	enabled := []control.ChangedField{control.ChangedFieldEnabled}
	for _, tc := range []struct {
		name  string
		cargo string
		want  quickBolusFields
	}{
		{"disable carrying one unit", "0000e803d00701", quickBolusFields{false, units, 1000, 2000, enabled}},
		{"disable carrying five units", "00008813d00701", quickBolusFields{false, units, 5000, 2000, enabled}},
		{"enable carrying one unit", "0100e803d00701", quickBolusFields{true, units, 1000, 2000, enabled}},
		{"enable at one unit", "0100e803d00705", quickBolusFields{true, units, 1000, 2000, []control.ChangedField{control.ChangedFieldEnabled, control.ChangedFieldIncrementUnits}}},
		{"increment half a unit", "0100f401d00704", quickBolusFields{true, units, 500, 2000, []control.ChangedField{control.ChangedFieldIncrementUnits}}},
		{"switch to carbs", "01018813d00702", quickBolusFields{true, carbs, 5000, 2000, []control.ChangedField{control.ChangedFieldMode}}},
		{"disable in carbs mode carrying the half unit capture", "0001f401d00701", quickBolusFields{false, carbs, 500, 2000, enabled}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.want.check(t, quickBolusRequestFromHex(t, tc.cargo))
			built := control.NewSetQuickBolusSettingsRequest(tc.want.enabled, tc.want.mode.Raw(), tc.want.incrementUnits, tc.want.incrementCarbs,
				control.ChangedFieldToBitmask(tc.want.changed...))
			if got := hex.EncodeToString(built.Cargo()); got != tc.cargo {
				t.Errorf("built cargo %s, want %s", got, tc.cargo)
			}
		})
	}
}

func TestSetQuickBolusSettingsRequestForChange(t *testing.T) {
	units, carbs := control.QuickBolusModeUnits, control.QuickBolusModeCarbs
	for _, tc := range []struct {
		name      string
		current   *currentstatus.PumpGlobalsResponse
		enabled   bool
		mode      control.QuickBolusMode
		increment int32
		cargo     string
	}{
		{"enabling at the current half unit", pumpGlobals(false, units, 500, 2000), true, units, 500, "0100f401d00701"},
		{"enabling at a new increment sends both fields", pumpGlobals(false, units, 500, 2000), true, units, 1000, "0100e803d00705"},
		{"changing to half a unit", pumpGlobals(true, units, 1000, 2000), true, units, 500, "0100f401d00704"},
		{"changing to five units", pumpGlobals(true, units, 2000, 2000), true, units, 5000, "01008813d00704"},
		{"disabling ignores mode and increment", pumpGlobals(true, units, 5000, 2000), false, carbs, 0, "00008813d00701"},
		{"disabling in carbs mode carries carbs mode", pumpGlobals(true, carbs, 5000, 10000), false, units, 0, "00018813102701"},
		{"switching to carbs at the current increment", pumpGlobals(true, units, 5000, 2000), true, carbs, 2000, "01018813d00702"},
		{"changing to five grams", pumpGlobals(true, carbs, 5000, 2000), true, carbs, 5000, "01018813881308"},
		{"switching back to units", pumpGlobals(true, carbs, 5000, 10000), true, units, 5000, "01008813102702"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := control.NewSetQuickBolusSettingsRequestForChange(tc.current, tc.enabled, tc.mode, tc.increment)
			if err != nil {
				t.Fatal(err)
			}
			if got := hex.EncodeToString(m.Cargo()); got != tc.cargo {
				t.Errorf("cargo %s, want %s", got, tc.cargo)
			}
		})
	}
}

func TestSetQuickBolusSettingsRequestForChangeChangesNothingWhenThePumpAlreadyMatches(t *testing.T) {
	m, err := control.NewSetQuickBolusSettingsRequestForChange(pumpGlobals(true, control.QuickBolusModeUnits, 2000, 2000), true, control.QuickBolusModeUnits, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if m.ChangedFieldsRaw != 0 {
		t.Errorf("changed fields %v", m.ChangedFields())
	}
}

func TestSetQuickBolusSettingsRequestForChangeRefusesAnUnsupportedIncrement(t *testing.T) {
	if _, err := control.NewSetQuickBolusSettingsRequestForChange(pumpGlobals(true, control.QuickBolusModeUnits, 500, 2000), true, control.QuickBolusModeUnits, 750); err == nil {
		t.Error("a 750 milliunit increment was built")
	}
	if _, err := control.NewSetQuickBolusSettingsRequestForChange(pumpGlobals(true, control.QuickBolusModeCarbs, 500, 2000), true, control.QuickBolusModeCarbs, 500); err == nil {
		t.Error("a 500 milligram increment was built")
	}
}

func TestSetQuickBolusSettingsRequestForIncrementChange(t *testing.T) {
	for _, tc := range []struct {
		current   *currentstatus.PumpGlobalsResponse
		increment control.QuickBolusIncrement
		cargo     string
	}{
		{pumpGlobals(false, control.QuickBolusModeUnits, 500, 2000), control.QuickBolusIncrementUnits1_0, "0100e803d00705"},
		{pumpGlobals(true, control.QuickBolusModeUnits, 5000, 2000), control.QuickBolusIncrementCarbs5G, "0101881388130a"},
		{pumpGlobals(true, control.QuickBolusModeUnits, 5000, 2000), control.QuickBolusIncrementDisabled, "00008813d00701"},
	} {
		m, err := control.NewSetQuickBolusSettingsRequestForIncrementChange(tc.current, tc.increment)
		if err != nil {
			t.Fatal(err)
		}
		if got := hex.EncodeToString(m.Cargo()); got != tc.cargo {
			t.Errorf("%v: cargo %s, want %s", tc.increment, got, tc.cargo)
		}
	}
}

func TestSetQuickBolusSettingsRequestForIncrementIsTheCapture(t *testing.T) {
	for _, tc := range []struct {
		increment control.QuickBolusIncrement
		cargo     string
	}{
		{control.QuickBolusIncrementDisabled, "0000f401d00701"},
		{control.QuickBolusIncrementUnits0_5, "0100f401d00701"},
		{control.QuickBolusIncrementUnits1_0, "0100e803d00704"},
		{control.QuickBolusIncrementUnits2_0, "0100d007d00704"},
		{control.QuickBolusIncrementUnits5_0, "01008813d00704"},
		{control.QuickBolusIncrementCarbs2G, "01018813d00708"},
		{control.QuickBolusIncrementCarbs5G, "01018813881308"},
		{control.QuickBolusIncrementCarbs10G, "01018813102708"},
		{control.QuickBolusIncrementCarbs15G, "01018813983a08"},
	} {
		got := control.NewSetQuickBolusSettingsRequestForIncrement(tc.increment)
		if want := quickBolusRequestFromHex(t, tc.cargo); !reflect.DeepEqual(got, want) {
			t.Errorf("%v: %s, want %s", tc.increment, messages.ToString(got), messages.ToString(want))
		}
		withMode, err := control.NewSetQuickBolusSettingsRequestForModeAndIncrement(tc.increment.Enabled(), tc.increment.Mode(), tc.increment)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(withMode, got) {
			t.Errorf("%v with its mode: %s, want %s", tc.increment, messages.ToString(withMode), messages.ToString(got))
		}
	}
}

func TestSetQuickBolusSettingsRequestRefusesAContradictoryEnabledFlag(t *testing.T) {
	if _, err := control.NewSetQuickBolusSettingsRequestForModeAndIncrement(true, control.QuickBolusModeUnits, control.QuickBolusIncrementDisabled); err == nil {
		t.Error("an enabled Disabled was built")
	}
	if _, err := control.NewSetQuickBolusSettingsRequestForModeAndIncrement(false, control.QuickBolusModeUnits, control.QuickBolusIncrementUnits1_0); err == nil {
		t.Error("a disabled Units1_0 was built")
	}
}

func TestSetQuickBolusSettingsRequestIncrementFollowsTheFieldsNotTheCapture(t *testing.T) {
	for _, tc := range []struct {
		cargo string
		want  control.QuickBolusIncrement
	}{
		{"0100e803d00701", control.QuickBolusIncrementUnits1_0},
		{"01018813d00702", control.QuickBolusIncrementCarbs2G},
		{"00018813102701", control.QuickBolusIncrementDisabled},
	} {
		if got := quickBolusRequestFromHex(t, tc.cargo).Increment(); got == nil || *got != tc.want {
			t.Errorf("%s: increment %v, want %v", tc.cargo, got, tc.want)
		}
	}
	if got := quickBolusRequestFromHex(t, "0100ee02d00704").Increment(); got != nil {
		t.Errorf("a 750 milliunit increment is %v", *got)
	}
}

func TestQuickBolusIncrementOfPumpGlobals(t *testing.T) {
	for _, tc := range []struct {
		globals *currentstatus.PumpGlobalsResponse
		want    control.QuickBolusIncrement
	}{
		{pumpGlobals(false, control.QuickBolusModeCarbs, 1000, 5000), control.QuickBolusIncrementDisabled},
		{pumpGlobals(true, control.QuickBolusModeUnits, 1000, 5000), control.QuickBolusIncrementUnits1_0},
		{pumpGlobals(true, control.QuickBolusModeCarbs, 1000, 5000), control.QuickBolusIncrementCarbs5G},
	} {
		if got := control.QuickBolusIncrementOf(tc.globals); got == nil || *got != tc.want {
			t.Errorf("increment %v, want %v", got, tc.want)
		}
	}
	if got := control.QuickBolusIncrementOf(pumpGlobals(true, control.QuickBolusModeUnits, 100, 1)); got != nil {
		t.Errorf("a 100 milliunit increment is %v", *got)
	}
	if got := control.QuickBolusModeOf(pumpGlobals(true, control.QuickBolusModeCarbs, 1000, 5000)); got == nil || *got != control.QuickBolusModeCarbs {
		t.Errorf("mode %v", got)
	}
}

func TestSetQuickBolusSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetQuickBolusSettingsRequest")
}
