package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestControlIQInfoV1Response(t *testing.T) {
	for _, tc := range []struct {
		name       string
		hex        string
		txID       int
		weight     int32
		weightUnit int32
	}{
		{"pounds", "000369030a019600014b0000000400b912", 3, 150, 1},
		{"kilograms", "000469040a018c00024b0000000400167e", 4, 140, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus,
				currentstatus.NewControlIQInfoV1Response(true, tc.weight, tc.weightUnit, 75, 0, 0, 0, 4, 0))
		})
	}
}

func TestControlIQInfoV1ResponseModeAndWeightUnit(t *testing.T) {
	m := currentstatus.NewControlIQInfoV1Response(true, 150, 1, 75, 1, 0, 0, 4, 0)
	if mode := m.CurrentUserModeType(); mode == nil || *mode != currentstatus.UserModeTypeSleep {
		t.Errorf("mode %v, want SLEEP", mode)
	}
	if unit := m.WeightUnit(); unit == nil || *unit != currentstatus.WeightUnitPounds {
		t.Errorf("weight unit %v, want POUNDS", unit)
	}
}

func TestControlIQInfoV1ResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ControlIQInfoV1Response")
}
