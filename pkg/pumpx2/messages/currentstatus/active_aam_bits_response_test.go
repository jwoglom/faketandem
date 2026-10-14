package currentstatus_test

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestActiveAamBitsResponseReadsMalfunctionBitmasks(t *testing.T) {
	built := currentstatus.NewActiveAamBitsResponse(big.NewInt(1<<9), big.NewInt(1<<12|1<<13), currentstatus.AamTypeMalfunction)
	parsed, err := currentstatus.NewActiveAamBitsResponseFromCargo(built.Cargo())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.AamType != currentstatus.AamTypeMalfunction {
		t.Errorf("type %s", parsed.AamType)
	}
	if !parsed.HasUnacknowledgedBit(9) || !parsed.HasActiveBit(12) || !parsed.HasActiveBit(13) {
		t.Errorf("unacknowledged %s, active %s", parsed.UnacknowledgedBitmask, parsed.ActiveBitmask)
	}
	want := []currentstatus.MalfunctionType{currentstatus.MalfunctionTypeVibe, currentstatus.MalfunctionTypePeriphPower}
	if !reflect.DeepEqual(parsed.ActiveMalfunctions, want) {
		t.Errorf("active malfunctions %v, want %v", parsed.ActiveMalfunctions, want)
	}
	if parsed.Size() != 2 {
		t.Errorf("size %d", parsed.Size())
	}
}

func TestActiveAamBitsResponseListsMalfunctionsOnlyForMalfunctions(t *testing.T) {
	parsed := currentstatus.NewActiveAamBitsResponse(big.NewInt(1<<9), big.NewInt(1<<12), currentstatus.AamTypeAlarm)
	if len(parsed.ActiveMalfunctions) != 0 || len(parsed.UnacknowledgedMalfunctions) != 0 || !parsed.HasActiveBit(12) {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestActiveAamBitsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ActiveAamBitsResponse")
}
