package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestHomeScreenMirrorResponse(t *testing.T) {
	expected := currentstatus.NewHomeScreenMirrorResponse(0, 255, 200, 200, 200, 4, 0, false, false)
	parsed := parity.TestMessage(t, "000339030900ffc8c8c804000000c1a1", 3, 1, messages.CurrentStatus,
		expected).(*currentstatus.HomeScreenMirrorResponse)
	for _, c := range []struct {
		name      string
		got, want interface{}
	}{
		{"CGM trend", *parsed.CgmTrendIcon, currentstatus.CGMTrendIconNoArrow},
		{"CGM alert", *parsed.CgmAlertIcon, currentstatus.CGMAlertIconNoError},
		{"status 0", *parsed.StatusIcon0, currentstatus.StatusIcon0HideIcon},
		{"status 1", *parsed.StatusIcon1, currentstatus.StatusIcon1HideIcon},
		{"bolus", *parsed.BolusStatusIcon, currentstatus.BolusStatusIconHideIcon},
		{"basal", *parsed.BasalStatusIcon, currentstatus.BasalStatusIconSuspend},
		{"Control-IQ", *parsed.ApControlStateIcon, currentstatus.ApControlStateIconStateGray},
	} {
		if c.got != c.want {
			t.Errorf("%s icon %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestHomeScreenMirrorResponseLeavesAnUnknownIconNil(t *testing.T) {
	m := currentstatus.NewHomeScreenMirrorResponse(8, 9, 2, 2, 4, 8, 4, true, true)
	if m.CgmTrendIcon != nil || m.CgmAlertIcon != nil || m.StatusIcon0 != nil || m.StatusIcon1 != nil ||
		m.BolusStatusIcon != nil || m.BasalStatusIcon != nil || m.ApControlStateIcon != nil {
		t.Errorf("%s", messages.ToString(m))
	}
}

func TestHomeScreenMirrorResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "HomeScreenMirrorResponse")
}
