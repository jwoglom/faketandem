package currentstatus_test

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestGetG6TransmitterHardwareInfoResponse(t *testing.T) {
	raw, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000" +
		"0000000000000000000000000000000000000000000000000000000000000000" +
		"3135373900000000000000000000000000000000000000000000000000000000")
	parsed, err := currentstatus.NewGetG6TransmitterHardwareInfoResponseFromCargo(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), raw) {
		t.Errorf("cargo %x", parsed.Cargo())
	}
	want := currentstatus.GetG6TransmitterHardwareInfoResponse{TransmitterPairingCode: "1579"}
	want.SetCargo(raw)
	if !reflect.DeepEqual(*parsed, want) {
		t.Errorf("parsed %+v, want %+v", *parsed, want)
	}
}

func TestGetG6TransmitterHardwareInfoResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "GetG6TransmitterHardwareInfoResponse")
}
