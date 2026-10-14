package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStreamDataReadinessRequestReadsAnyOtherOrdinalAsUnknown(t *testing.T) {
	m := &currentstatus.StreamDataReadinessRequest{}
	if err := m.Parse([]byte{0xc7}); err != nil {
		t.Fatal(err)
	}
	if m.StreamDataTypeID != 199 || m.StreamDataType != currentstatus.StreamDataTypeUnknown {
		t.Errorf("type %d %v, want 199 UNKNOWN", m.StreamDataTypeID, m.StreamDataType)
	}
	fsl2 := currentstatus.NewStreamDataReadinessRequest(currentstatus.StreamDataTypeFreestyleLibre2SensorContext)
	if fsl2.StreamDataTypeID != 0 || fsl2.Cargo()[0] != 0 {
		t.Errorf("FSL2 sensor context is %d, cargo %x", fsl2.StreamDataTypeID, fsl2.Cargo())
	}
}

func TestStreamDataReadinessRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StreamDataReadinessRequest")
}
