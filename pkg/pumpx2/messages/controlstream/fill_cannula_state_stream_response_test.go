package controlstream_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/controlstream"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestFillCannulaStateStreamResponse(t *testing.T) {
	expected := controlstream.NewFillCannulaStateStreamResponse(2)
	parsed := testCapture(t, "0070e7701902e7d64020c63c1d5e2cfb1fecb965df1f825d770093d1d6461306", 0x70,
		expected).(*controlstream.FillCannulaStateStreamResponse)
	if parsed.State == nil || *parsed.State != controlstream.FillCannulaStateCannulaFilled {
		t.Errorf("state %v, want CANNULA_FILLED", parsed.State)
	}
}

func TestFillCannulaStateStreamResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "FillCannulaStateStreamResponse")
}
