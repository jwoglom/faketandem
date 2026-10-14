package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBasalIQAlertInfoRequest(t *testing.T) {
	parity.TestMessage(t, "000366030004b0", 3, 1, messages.CurrentStatus, currentstatus.NewBasalIQAlertInfoRequest())
}

func TestBasalIQAlertInfoRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BasalIQAlertInfoRequest")
}
