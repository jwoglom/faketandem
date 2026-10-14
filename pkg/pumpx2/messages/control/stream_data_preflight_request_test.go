package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestStreamDataPreflightRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "StreamDataPreflightRequest")
}
