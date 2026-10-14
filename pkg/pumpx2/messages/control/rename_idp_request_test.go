package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestRenameIDPRequest(t *testing.T) {
	expected, err := control.NewRenameIDPRequest(1, 1, "testprofil2")
	if err != nil {
		t.Fatal(err)
	}
	parity.TestMessage(t, "02d6a8d62b01017465737470726f66696c320000", -42, 1, messages.Control, expected,
		"01d600000000e4483e2065ee618e7bd6a980f0a6", "00d60656ac383c1140bccf5a20be")
}

func TestRenameIDPRequestRefusesANameLongerThan16Bytes(t *testing.T) {
	if _, err := control.NewRenameIDPRequest(1, 1, "seventeen chars!!"); err == nil {
		t.Error("a 17-byte name was built")
	}
}

func TestRenameIDPRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "RenameIDPRequest")
}
