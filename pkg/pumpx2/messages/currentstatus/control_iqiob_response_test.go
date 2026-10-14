package currentstatus_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestControlIQIOBResponse(t *testing.T) {
	for _, tc := range []struct {
		name          string
		hex           string
		txID          int
		want          *currentstatus.ControlIQIOBResponse
		wantDisplayed int64
	}{
		{"no insulin on board", "00036d031100000000000000000000000000000000015aa3", 3,
			currentstatus.NewControlIQIOBResponse(0, 0, 0, 0, currentstatus.IOBTypeSwan6hr.ID()), 0},
		{"Control-IQ off", "00056d0511c00000007c380000f1000000a100000000ab8d", 5,
			currentstatus.NewControlIQIOBResponse(192, 14460, 241, 161, currentstatus.IOBTypeMudaliar.ID()), 192},
		{"Control-IQ on", "00066d0611b80000008c370000f10000009a00000001c315", 6,
			currentstatus.NewControlIQIOBResponse(184, 14220, 241, 154, currentstatus.IOBTypeSwan6hr.ID()), 154},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 2, messages.CurrentStatus, tc.want).(*currentstatus.ControlIQIOBResponse)
			if !reflect.DeepEqual(parsed, tc.want) {
				t.Errorf("parsed %+v, want %+v", parsed, tc.want)
			}
			if got := parsed.PumpDisplayedIOB(); got != tc.wantDisplayed {
				t.Errorf("displayed IOB %d, want %d", got, tc.wantDisplayed)
			}
		})
	}
}

func TestControlIQIOBResponseDisplaysTheLargerIOBForAnUnknownType(t *testing.T) {
	m := currentstatus.NewControlIQIOBResponse(150, 0, 200, 300, 2)
	if m.IOBType() != nil {
		t.Errorf("IOB type %v, want nil", m.IOBType())
	}
	if got := m.PumpDisplayedIOB(); got != 300 {
		t.Errorf("displayed IOB %d, want 300", got)
	}
}

func TestControlIQIOBResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ControlIQIOBResponse")
}
