package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestDismissNotificationResponse(t *testing.T) {
	parity.TestMessage(t, "001eb91e19003ba8f21fe16f0c1b71446a28fadaa20c0304576230243bfe5a5d", 30, 1, messages.Control, control.NewDismissNotificationResponse(0))
}

func TestDismissNotificationResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "DismissNotificationResponse")
}
