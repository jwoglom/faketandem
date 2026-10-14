package currentstatus_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPumpSettingsResponse(t *testing.T) {
	parity.TestMessage(t, "0003530309231e01120000785d00703f", 3, 1, messages.CurrentStatus,
		currentstatus.NewPumpSettingsResponse(35, 30, 1, 18, 0, 120, 93))
}

func TestPumpSettingsResponseOfAMobi(t *testing.T) {
	expected := currentstatus.NewPumpSettingsResponse(16, 10, 0, 5, 0, 30, 15)
	parsed, err := currentstatus.NewPumpSettingsResponseFromCargo([]byte{16, 10, 0, 5, 0, 0, 30, 15, 0})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed, expected) {
		t.Errorf("parsed %+v, want %+v", *parsed, *expected)
	}
}

func TestPumpSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PumpSettingsResponse")
}
