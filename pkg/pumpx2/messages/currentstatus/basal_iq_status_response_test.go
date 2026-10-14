package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBasalIQStatusResponseNamesItsState(t *testing.T) {
	if s := currentstatus.NewBasalIQStatusResponse(1, false).State(); s == nil || *s != currentstatus.BasalIQStatusStateSuspend {
		t.Errorf("state %v", s)
	}
	if s := currentstatus.NewBasalIQStatusResponse(4, true).State(); s != nil {
		t.Errorf("state %v for an unknown ID", s)
	}
}

func TestBasalIQStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BasalIQStatusResponse")
}
