package currentstatus_test

import (
	"bytes"
	"math/big"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestMalfunctionBitmaskStatusResponseReadsBitmask(t *testing.T) {
	bitmask := big.NewInt(1 | 1<<13 | 1<<25)
	response := currentstatus.NewMalfunctionBitmaskStatusResponse(bitmask)
	if response.Bitmask.Cmp(bitmask) != 0 {
		t.Errorf("bitmask %s, want %s", response.Bitmask, bitmask)
	}
	if !response.HasMalfunctionBit(0) || !response.HasMalfunctionBit(13) || !response.HasMalfunctionBit(25) {
		t.Errorf("bits of %s", response.Bitmask)
	}
	want := []currentstatus.MalfunctionType{
		currentstatus.MalfunctionTypeSoftware, currentstatus.MalfunctionTypePeriphPower, currentstatus.MalfunctionTypePushoffStall,
	}
	if !reflect.DeepEqual(response.Malfunctions, want) {
		t.Errorf("malfunctions %v, want %v", response.Malfunctions, want)
	}
	if !response.HasMalfunction(currentstatus.MalfunctionTypePushoffStall) || response.Size() != 3 {
		t.Errorf("size %d", response.Size())
	}
}

func TestMalfunctionBitmaskStatusResponseParsesZeroBitmask(t *testing.T) {
	expected := currentstatus.NewMalfunctionBitmaskStatusResponse(big.NewInt(0))
	parsed := &currentstatus.MalfunctionBitmaskStatusResponse{}
	if err := parsed.Parse(make([]byte, 8)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	if parsed.Bitmask.Sign() != 0 || len(parsed.Malfunctions) != 0 || parsed.Size() != 0 {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestMalfunctionBitmaskStatusResponseFromCodes(t *testing.T) {
	response := currentstatus.NewMalfunctionBitmaskStatusResponseFromCodes(1<<4, 1)
	if response.CodeA() != 1<<4 || response.CodeB() != 1 || !response.HasMalfunctionBit(4) || !response.HasMalfunctionBit(32) {
		t.Errorf("codes %d %d, bitmask %s", response.CodeA(), response.CodeB(), response.Bitmask)
	}
}

func TestMalfunctionBitmaskStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "MalfunctionBitmaskStatusResponse")
}
