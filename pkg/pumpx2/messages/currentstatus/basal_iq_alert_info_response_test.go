package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBasalIQAlertInfoResponseNamesItsAlert(t *testing.T) {
	if a := currentstatus.NewBasalIQAlertInfoResponse(24577).Alert(); a == nil || *a != currentstatus.BasalIQAlertInsulinResumedAlert {
		t.Errorf("alert %v", a)
	}
	if a := currentstatus.NewBasalIQAlertInfoResponse(1).Alert(); a != nil {
		t.Errorf("alert %v for an unknown ID", a)
	}
}

func TestBasalIQAlertInfoResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BasalIQAlertInfoResponse")
}
