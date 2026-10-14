package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetSiteChangeReminderRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetSiteChangeReminderRequest")
}
