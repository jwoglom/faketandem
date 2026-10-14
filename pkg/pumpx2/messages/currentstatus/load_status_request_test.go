package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLoadStatusRequestHasEmptyCargo(t *testing.T) {
	m := currentstatus.NewLoadStatusRequest()
	if len(m.Cargo()) != 0 {
		t.Errorf("cargo %x", m.Cargo())
	}
	if err := m.Parse([]byte{}); err != nil {
		t.Fatal(err)
	}
	if len(m.Cargo()) != 0 {
		t.Errorf("parsed cargo %x", m.Cargo())
	}
}

func TestLoadStatusRequestRefusesAPayload(t *testing.T) {
	if err := currentstatus.NewLoadStatusRequest().Parse([]byte{1}); err == nil {
		t.Error("a one-byte payload parsed")
	}
}

func TestLoadStatusRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LoadStatusRequest")
}
