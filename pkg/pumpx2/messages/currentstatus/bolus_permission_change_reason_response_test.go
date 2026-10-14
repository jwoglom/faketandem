package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBolusPermissionChangeReasonResponseNamesItsReason(t *testing.T) {
	m := currentstatus.NewBolusPermissionChangeReasonResponse(10, true, 3, false)
	if r := m.LastChangeReason(); r == nil || *r != currentstatus.ChangeReasonRevokedTimeout {
		t.Errorf("reason %v", r)
	}
}

func TestBolusPermissionChangeReasonResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BolusPermissionChangeReasonResponse")
}
