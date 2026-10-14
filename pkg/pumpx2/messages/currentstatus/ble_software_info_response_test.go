package currentstatus_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBleSoftwareInfoResponse(t *testing.T) {
	expected := currentstatus.NewBleSoftwareInfoResponse(132, 7, 2, 0, 10, 257)
	raw, _ := hex.DecodeString("84000700020000000a0000000101")
	parsed, err := currentstatus.NewBleSoftwareInfoResponseFromCargo(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	if parsed.SoftDeviceID != 132 || parsed.SoftDeviceMajorVersion != 7 || parsed.SoftDeviceMinorVersion != 2 ||
		parsed.SoftDeviceBugfixVersion != 0 || parsed.SoftDeviceVersion != 10 || parsed.SoftDeviceSubVersion != 257 {
		t.Errorf("parsed %+v", parsed)
	}
}

func TestBleSoftwareInfoResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BleSoftwareInfoResponse")
}
