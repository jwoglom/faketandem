package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStatusCodeFromCodeFallsBackToUnknown(t *testing.T) {
	for code, want := range map[int32]control.StatusCode{
		0: control.StatusCodeAck, 2: control.StatusCodeInvalidFeatureCommand, 3: control.StatusCodeUnknown, 255: control.StatusCodeUnknown,
	} {
		if got := control.StatusCodeFromCode(code); got != want {
			t.Errorf("code %d: %v, want %v", code, got, want)
		}
	}
}

func TestSendTipsControlGenericTestResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SendTipsControlGenericTestResponse")
}
