package currentstatus_test

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestExtendedBolusStatusV2Response(t *testing.T) {
	raw, _ := hex.DecodeString("00000022e700000000000000000000000000ffffffff")
	parsed := &currentstatus.ExtendedBolusStatusV2Response{}
	if err := parsed.Parse(raw); err != nil {
		t.Fatal(err)
	}
	want := currentstatus.ExtendedBolusStatusV2Response{SecondsSincePumpReset: 4294967295}
	want.SetCargo(raw)
	if !reflect.DeepEqual(*parsed, want) {
		t.Errorf("parsed %+v, want %+v", *parsed, want)
	}
}

func TestExtendedBolusStatusV2ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ExtendedBolusStatusV2Response")
}
