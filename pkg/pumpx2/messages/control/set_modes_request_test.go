package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetModesRequest(t *testing.T) {
	tester := parity.MessageTester{Env: messages.Env{TimeSinceReset: "1905716"}}
	for _, tc := range []struct {
		name    string
		command control.ModeCommand
		packets []string
		txID    int
	}{
		{"exercise mode on", control.ModeCommandExerciseModeOn, []string{"0179cc7919034d978b1e32c70c65d5b530fdf45a", "0079b59d63358bab1b849bb1a4e4"}, 121},
		{"exercise mode off", control.ModeCommandExerciseModeOff, []string{"0181cc81190461978b1eff1bb7778cd232419aa1", "0081d97936364e86562320d973e2"}, -127},
		{"sleep mode on", control.ModeCommandSleepModeOn, []string{"018acc8a19017d978b1ef95a55180db9a2f1c821", "008ace22b462fab923c16c2cc428"}, -118},
		{"sleep mode off", control.ModeCommandSleepModeOff, []string{"0191cc9119029a978b1e7ce08a7a68802cca71dc", "009134fd2877d2a8bd3247ef819d"}, -111},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := control.NewSetModesRequestFromCargo([]byte{byte(tc.command)})
			if err != nil {
				t.Fatal(err)
			}
			parsed := tester.Test(t, tc.packets[0], tc.txID, 2, messages.Control, expected, tc.packets[1:]...).(*control.SetModesRequest)
			if parsed.Bitmap != tc.command.Bitmap() || parsed.Command == nil || *parsed.Command != tc.command {
				t.Errorf("parsed %+v", parsed)
			}
		})
	}
}

func TestSetModesRequestForACommand(t *testing.T) {
	m := control.NewSetModesRequest(control.ModeCommandSleepModeOn)
	if m.Bitmap != 1 || m.Command == nil || *m.Command != control.ModeCommandSleepModeOn || len(m.Cargo()) != 1 || m.Cargo()[0] != 1 {
		t.Errorf("built %+v", m)
	}
	if m := control.NewSetModesRequestFromBitmap(9); m.Command != nil {
		t.Errorf("bitmap 9 is %v", *m.Command)
	}
}

func TestSetModesRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetModesRequest")
}
