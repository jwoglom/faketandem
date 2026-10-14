package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetMaxBolusLimitRequest(t *testing.T) {
	expected, err := control.NewSetMaxBolusLimitRequest(20000)
	if err != nil {
		t.Fatal(err)
	}
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1"}}
	tester.Test(t, "016186611a204e92e14920c2f4aee8ca1b7d1960", 97, 1, messages.Control, expected, "006175ffd49542eeb8c009ed2022f2")
}

func TestSetMaxBolusLimitRequestRefusesALimitOutsideOneToTwentyFiveUnits(t *testing.T) {
	for _, limit := range []int32{500, 25100} {
		if _, err := control.NewSetMaxBolusLimitRequest(limit); err == nil {
			t.Errorf("a %d milliunit limit was built", limit)
		}
	}
}

func TestSetMaxBolusLimitRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetMaxBolusLimitRequest")
}
