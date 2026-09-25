package handler

import (
	"encoding/json"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2"
)

// cliparser's output parser turns integral fields into Go ints, so handlers that
// assert .(float64) against them silently read nothing. A cargo map that has
// been through JSON (the HTTP/websocket API) holds float64s instead. Both must
// work, along with the first-key-wins fallback that lets a handler accept an
// alternative spelling.
func TestCargoInt_AcceptsEveryNumericRepresentation(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{"int, as cliparser output produces", 2500},
		{"int64", int64(2500)},
		{"uint32", uint32(2500)},
		{"float64, as a JSON round trip produces", float64(2500)},
		{"json.Number", json.Number("2500")},
		{"numeric string", "2500"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &pumpx2.ParsedMessage{Cargo: map[string]interface{}{"totalVolume": tc.value}}
			got, ok := cargoInt(msg, "totalVolume")
			if !ok {
				t.Fatalf("cargoInt did not read a %T", tc.value)
			}
			if got != 2500 {
				t.Errorf("cargoInt = %d, want 2500", got)
			}
		})
	}
}

func TestCargoInt_MissingAndNonNumeric(t *testing.T) {
	msg := &pumpx2.ParsedMessage{Cargo: map[string]interface{}{"other": 1, "name": "MANUAL"}}

	if _, ok := cargoInt(msg, "totalVolume"); ok {
		t.Error("cargoInt reported success for an absent key")
	}
	if _, ok := cargoInt(msg, "name"); ok {
		t.Error("cargoInt reported success for a non-numeric string")
	}
	if _, ok := cargoInt(nil, "totalVolume"); ok {
		t.Error("cargoInt reported success for a nil message")
	}
}

func TestCargoInt_FirstPresentKeyWins(t *testing.T) {
	msg := &pumpx2.ParsedMessage{Cargo: map[string]interface{}{"percentage": 100, "percent": 150}}

	got, ok := cargoInt(msg, "percent", "percentage")
	if !ok || got != 150 {
		t.Errorf("cargoInt = %d (ok=%v), want the first listed key's value 150", got, ok)
	}
}

func TestCargoBool(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"Go bool", true, true},
		{"string, as cliparser prints Java booleans", "true", true},
		{"string false", "false", false},
		{"numeric 1", 1, true},
		{"numeric 0", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &pumpx2.ParsedMessage{Cargo: map[string]interface{}{"active": tc.value}}
			got, ok := cargoBool(msg, "active")
			if !ok {
				t.Fatalf("cargoBool did not read a %T", tc.value)
			}
			if got != tc.want {
				t.Errorf("cargoBool = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSetTempRateResponseRawCargo(t *testing.T) {
	// [status][tempRateId uint16 little-endian][reserved], 4 bytes, matching
	// pumpX2's declared size and its own parse().
	if got := setTempRateResponseRawCargo(0, 1); got != "00010000" {
		t.Errorf("setTempRateResponseRawCargo(0, 1) = %q, want \"00010000\"", got)
	}
	if got := setTempRateResponseRawCargo(0, 258); got != "00020100" {
		t.Errorf("setTempRateResponseRawCargo(0, 258) = %q, want \"00020100\" (0x0102 little-endian)", got)
	}
}
