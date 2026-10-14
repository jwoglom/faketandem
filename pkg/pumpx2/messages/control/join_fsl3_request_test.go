package control_test

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestJoinFSL3Request(t *testing.T) {
	decode := func(s string) []byte {
		b, err := hex.DecodeString(s)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	bleAddress, sensorCode, activationTime := decode("010203040506"), decode("0a0b0c0d"), decode("11223344")
	firmwareVersion, compressedSerialNumber, nfcUID := decode("01000200"), decode("112233445566778899"), decode("a1a2a3a4a5a6a7a8")
	expected, err := control.NewJoinFSL3Request(bleAddress, sensorCode, activationTime, firmwareVersion, 60, compressedSerialNumber, nfcUID)
	if err != nil {
		t.Fatal(err)
	}
	parsed := &control.JoinFSL3Request{}
	if err := parsed.Parse(expected.Cargo()); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Cargo(), expected.Cargo()) || len(parsed.Cargo()) != 36 {
		t.Errorf("cargo %x, want %x", parsed.Cargo(), expected.Cargo())
	}
	for _, f := range []struct {
		name      string
		got, want []byte
	}{
		{"bleAddress", parsed.BLEAddress, bleAddress},
		{"sensorCode", parsed.SensorCode, sensorCode},
		{"activationTime", parsed.ActivationTime, activationTime},
		{"firmwareVersion", parsed.FirmwareVersion, firmwareVersion},
		{"compressedSerialNumber", parsed.CompressedSerialNumber, compressedSerialNumber},
		{"nfcUid", parsed.NFCUID, nfcUID},
	} {
		if !bytes.Equal(f.got, f.want) {
			t.Errorf("%s %x, want %x", f.name, f.got, f.want)
		}
	}
	if parsed.WarmupTime != 60 {
		t.Errorf("warmupTime %d, want 60", parsed.WarmupTime)
	}
}

func TestJoinFSL3RequestRefusesAFieldOfTheWrongLength(t *testing.T) {
	four := make([]byte, 4)
	if _, err := control.NewJoinFSL3Request(make([]byte, 5), four, four, four, 0, make([]byte, 9), make([]byte, 8)); err == nil {
		t.Error("a 5-byte BLE address was built")
	}
}

func TestJoinFSL3RequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "JoinFSL3Request")
}
