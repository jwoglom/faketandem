package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBasalIQSettingsResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BasalIQSettingsResponse")
}
