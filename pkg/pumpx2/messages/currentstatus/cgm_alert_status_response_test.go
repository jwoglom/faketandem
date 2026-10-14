package currentstatus_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestCGMAlertStatusResponse(t *testing.T) {
	parsed := parity.TestMessage(t, "00034b030800000000000000009ed2", 3, 1, messages.CurrentStatus,
		currentstatus.NewCGMAlertStatusResponse(0)).(*currentstatus.CGMAlertStatusResponse)
	if parsed.Size() != 0 {
		t.Errorf("alerts %v", parsed.CGMAlerts())
	}
}

func TestCGMAlertStatusResponseListsAlertsAndKnownIDs(t *testing.T) {
	m := currentstatus.NewCGMAlertStatusResponse(1<<int(currentstatus.CGMAlertHighCGMAlert) | 1<<int(currentstatus.CGMAlertDefaultCGMAlert15))
	want := []currentstatus.CGMAlert{currentstatus.CGMAlertHighCGMAlert, currentstatus.CGMAlertDefaultCGMAlert15}
	if !reflect.DeepEqual(m.CGMAlerts(), want) {
		t.Errorf("alerts %v, want %v", m.CGMAlerts(), want)
	}
	if ids := m.NotificationIDs(); !reflect.DeepEqual(ids, []int32{2}) {
		t.Errorf("notification IDs %v", ids)
	}
}

func TestCGMAlertStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "CGMAlertStatusResponse")
}
