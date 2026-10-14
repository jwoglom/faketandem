package control_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetSensorTypeRequestCargo(t *testing.T) {
	for _, id := range []int32{3, 0} {
		if got := control.NewSetSensorTypeRequestFromID(id).Cargo(); !bytes.Equal(got, []byte{byte(id)}) {
			t.Errorf("sensor type %d: cargo %x", id, got)
		}
	}
}

func TestSetSensorTypeRequestFromAnID(t *testing.T) {
	m := control.NewSetSensorTypeRequestFromID(1)
	if m.CgmSensorType != 1 || m.SensorType() != currentstatus.CgmStatusV2SensorTypeDexcomG6 {
		t.Errorf("built %+v (%v)", m, m.SensorType())
	}
}

func TestSetSensorTypeRequestFromASensorType(t *testing.T) {
	m := control.NewSetSensorTypeRequest(currentstatus.CgmStatusV2SensorTypeDexcomG7)
	if m.CgmSensorType != 3 || m.SensorType() != currentstatus.CgmStatusV2SensorTypeDexcomG7 {
		t.Errorf("built %+v (%v)", m, m.SensorType())
	}
}

func TestSetSensorTypeRequestParse(t *testing.T) {
	m := &control.SetSensorTypeRequest{}
	if err := m.Parse([]byte{2}); err != nil {
		t.Fatal(err)
	}
	if m.CgmSensorType != 2 || m.SensorType() != currentstatus.CgmStatusV2SensorTypeFSL2 {
		t.Errorf("parsed %+v (%v)", m, m.SensorType())
	}
}

func TestSetSensorTypeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetSensorTypeRequest")
}
