package control_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetSensorTypeResponseAck(t *testing.T) {
	m := control.NewSetSensorTypeResponse(0, 0)
	if !bytes.Equal(m.Cargo(), []byte{0, 0}) {
		t.Errorf("cargo %x", m.Cargo())
	}
	if m.Status != 0 || m.StatusAcknowledgement != 0 || m.StatusAcknowledgementType() != control.StatusAcknowledgementSuccess {
		t.Errorf("built %+v", m)
	}
}

func TestSetSensorTypeResponseNack(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cargo []byte
		want  control.StatusAcknowledgement
	}{
		{"invalid CGM type", []byte{1, 2}, control.StatusAcknowledgementInvalidCgmType},
		{"missing CGM support package", []byte{1, 4}, control.StatusAcknowledgementMissingOrInvalidCgmSupportPackage},
		{"an unknown code", []byte{1, 0x63}, control.StatusAcknowledgementUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &control.SetSensorTypeResponse{}
			if err := m.Parse(tc.cargo); err != nil {
				t.Fatal(err)
			}
			if m.Status != 1 || m.StatusAcknowledgement != int32(tc.cargo[1]) || m.StatusAcknowledgementType() != tc.want {
				t.Errorf("parsed %+v (%v), want %v", m, m.StatusAcknowledgementType(), tc.want)
			}
		})
	}
}

func TestSetSensorTypeResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetSensorTypeResponse")
}
