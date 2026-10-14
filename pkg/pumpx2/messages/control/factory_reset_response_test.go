package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestFactoryResetResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "FactoryResetResponse")
}
