package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSecretMenuResponse(t *testing.T) {
	parsed := parity.TestMessage(t, "0017bd17081caa342200001e00d577", 23, 1, messages.CurrentStatus,
		currentstatus.NewSecretMenuResponse(573876764, 1966080)).(*currentstatus.SecretMenuResponse)
	if parsed.TimeOfLastConnectionTimestampSeconds != 573876764 || parsed.ReservedValue != 1966080 {
		t.Errorf("last connection %d, reserved %d; want 573876764, 1966080", parsed.TimeOfLastConnectionTimestampSeconds, parsed.ReservedValue)
	}
}

func TestSecretMenuResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SecretMenuResponse")
}
