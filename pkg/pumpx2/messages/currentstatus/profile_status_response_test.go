package currentstatus_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestProfileStatusResponse(t *testing.T) {
	for _, tc := range []struct {
		name          string
		hex           string
		txID          int
		count         int32
		slot0, slot1  int32
		activeSegment int32
		slots         []int32
	}{
		{"one profile", "00033f03080100ffffffffff03d4c6", 3, 1, 0, -1, 3, []int32{0}},
		{"two profiles, the first active", "00043f0408020001ffffffff034403", 4, 2, 0, 1, 3, []int32{0, 1}},
		{"two profiles, the second active", "00113f1108020100ffffffff012188", 17, 2, 1, 0, 1, []int32{1, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := currentstatus.NewProfileStatusResponse(tc.count, tc.slot0, tc.slot1, -1, -1, -1, -1, tc.activeSegment)
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus, expected).(*currentstatus.ProfileStatusResponse)
			if got := parsed.IDPSlotIDs(); !reflect.DeepEqual(got, tc.slots) {
				t.Errorf("slots %v, want %v", got, tc.slots)
			}
		})
	}
}

func TestProfileStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ProfileStatusResponse")
}
