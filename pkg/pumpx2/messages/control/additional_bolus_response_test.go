package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestAdditionalBolusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "AdditionalBolusResponse")
}
