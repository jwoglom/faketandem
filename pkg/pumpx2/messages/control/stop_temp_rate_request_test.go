package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStopTempRateRequest(t *testing.T) {
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1906112"}}
	for _, tc := range []struct {
		name    string
		packets []string
		txID    int
	}{
		{"capture 1", []string{"012ca62c182c998b1e92605660ce66863620411d", "002c0d420615066cb0f317766c"}, 44},
		{"capture 2", []string{"0151a651181eeaee1ffbff278a74c8692c24136b", "00513e473b93df7f77e6fb867d"}, 81},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tester.Test(t, tc.packets[0], tc.txID, 2, messages.Control, control.NewStopTempRateRequest(), tc.packets[1:]...)
		})
	}
}

func TestStopTempRateRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StopTempRateRequest")
}
