package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestPlaySoundRequest(t *testing.T) {
	parity.TestMessage(t, "01cef4ce182337f31f36da5eea5ed250773df9b0", -50, 1, messages.Control, control.NewPlaySoundRequest(), "00ce91daca790983cb56d5fff1")
}

func TestPlaySoundRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "PlaySoundRequest")
}
