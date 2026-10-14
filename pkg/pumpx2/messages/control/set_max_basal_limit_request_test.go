package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetMaxBasalLimitRequest(t *testing.T) {
	expected, err := control.NewSetMaxBasalLimitRequest(5000)
	if err != nil {
		t.Fatal(err)
	}
	parity.TestMessage(t, "017588751c881300003ee34920d580d42e829ec9", 117, 1, messages.Control, expected,
		"00754d9fdc7a1f65f92b9b1f7c227bbb56")
}

func TestSetMaxBasalLimitRequestRefusesALimitOutsideOneToFifteenUnits(t *testing.T) {
	for _, limit := range []int64{100, 16000} {
		if _, err := control.NewSetMaxBasalLimitRequest(limit); err == nil {
			t.Errorf("a %d milliunit limit was built", limit)
		}
	}
}

func TestSetMaxBasalLimitRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetMaxBasalLimitRequest")
}
