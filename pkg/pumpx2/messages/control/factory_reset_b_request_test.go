package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestFactoryResetBRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "FactoryResetBRequest")
}
