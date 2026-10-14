package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetG6TransmitterIDRequest(t *testing.T) {
	expected, err := control.NewSetG6TransmitterIDRequest("8L459R")
	if err != nil {
		t.Fatal(err)
	}
	parsed := parity.TestMessage(t, "0221b02128384c34353952000000000000000000", 33, 1, messages.Control, expected,
		"012100da09f31f74a5fbd08dc7fc210ddd3cb101", "00218f1dc87134edb724ae").(*control.SetG6TransmitterIDRequest)
	if parsed.TxID != "8L459R" {
		t.Errorf("transmitter ID %q, want 8L459R", parsed.TxID)
	}
}

func TestSetG6TransmitterIDRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetG6TransmitterIdRequest")
}
