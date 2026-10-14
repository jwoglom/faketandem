package currentstatus_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLastBolusStatusV3ResponseWithNoBolus(t *testing.T) {
	expected := currentstatus.NewLastBolusStatusV3Response(
		0,
		0, 0, []byte{0, 0}, 0, 0, 0, 0, 0, 0, 4294967295,
		0, 0, []byte{0, 0}, 0, 0, 0, 0, 0, 0, 4294967295, 0,
	)
	raw, _ := hex.DecodeString("000000000000000000000000000000000000000000ffffffff0000000000000000000000000000000000000000ffffffff00000000")
	parsed := &currentstatus.LastBolusStatusV3Response{}
	if err := parsed.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	if !reflect.DeepEqual(parsed, expected) {
		t.Errorf("parsed %+v, want %+v", parsed, expected)
	}
	if parsed.IsStandardBolusPresent() || parsed.IsExtendedBolusPresent() {
		t.Error("a bolus is present")
	}
}

func TestLastBolusStatusV3ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LastBolusStatusV3Response")
}
