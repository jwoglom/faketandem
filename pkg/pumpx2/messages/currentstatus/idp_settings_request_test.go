package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestIDPSettingsRequest(t *testing.T) {
	parity.TestMessage(t, "000440040100ad05", 4, 1, messages.CurrentStatus, currentstatus.NewIDPSettingsRequest(0))
}

func TestIDPSettingsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "IDPSettingsRequest")
}
