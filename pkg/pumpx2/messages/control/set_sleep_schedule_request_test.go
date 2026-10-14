package control_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/models"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

// Schedule 1 every day from 00:00 to 23:59, turned on and then off.
func TestSetSleepScheduleRequest(t *testing.T) {
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1905836"}}
	for _, tc := range []struct {
		name    string
		enabled int32
		packets []string
		txID    int
	}{
		{"on", 1, []string{"02a2cea22000017f00009f0503ff978b1e828836", "01a222d3b77a25443cc291be989dc841df1a8a75", "00a266"}, -94},
		{"off", 0, []string{"02abceab2000007f00009f050311988b1ef9651b", "01abc15275490acd94a3416d8cfc038cd7ff461a", "00ab0c"}, -85},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := control.NewSetSleepScheduleRequestFromCargo([]byte{0, byte(tc.enabled), 127, 0, 0, 0x9f, 5, 3})
			if err != nil {
				t.Fatal(err)
			}
			parsed := tester.Test(t, tc.packets[0], tc.txID, 3, messages.Control, expected, tc.packets[1:]...).(*control.SetSleepScheduleRequest)
			s := parsed.SleepSchedule()
			if parsed.Slot != 0 || parsed.Flag != 3 || s.Enabled != tc.enabled {
				t.Errorf("parsed %+v", parsed)
			}
			if !reflect.DeepEqual(s.Days(), models.MultiDayFromBitmask(127)) {
				t.Errorf("days %v", s.Days())
			}
			if s.Start() != models.NewMinsTime(0) || s.End() != (models.MinsTime{Hour: 23, Min: 59}) {
				t.Errorf("from %v to %v", s.Start(), s.End())
			}
		})
	}
}

func TestSetSleepScheduleRequestFromASchedule(t *testing.T) {
	parsed, err := control.NewSetSleepScheduleRequestFromCargo([]byte{0, 1, 127, 0, 0, 0x9f, 5, 3})
	if err != nil {
		t.Fatal(err)
	}
	built := control.NewSetSleepScheduleRequest(0, parsed.SleepSchedule(), 3)
	if !reflect.DeepEqual(built.Cargo(), parsed.Cargo()) {
		t.Errorf("cargo %x, want %x", built.Cargo(), parsed.Cargo())
	}
}

func TestSetSleepScheduleRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetSleepScheduleRequest")
}
