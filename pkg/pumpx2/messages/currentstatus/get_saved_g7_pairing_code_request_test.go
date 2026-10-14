package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestGetSavedG7PairingCodeRequest(t *testing.T) {
	parity.TestMessage(t, "00247424007602", 36, 1, messages.CurrentStatus, currentstatus.NewGetSavedG7PairingCodeRequest())
}

func TestGetSavedG7PairingCodeRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "GetSavedG7PairingCodeRequest")
}
