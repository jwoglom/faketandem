package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLocalizationRequest(t *testing.T) {
	parity.TestMessage(t, "0004a60400640f", 4, 1, messages.CurrentStatus, currentstatus.NewLocalizationRequest())
}

func TestLocalizationRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LocalizationRequest")
}
