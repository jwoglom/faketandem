package control_test

import (
	"reflect"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestSetPumpSoundsRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cargo   []byte
		packets []string
		txID    int
	}{
		{"all beep", []byte{0, 0, 0, 0, 0, 0, 0, 2, 0xb6},
			[]string{"0282e482210000000000000002b68abd68206095", "0182305a4bf21a40e5554d391ed6f3eaa0d63126", "00829bd3"}, -126},
		{"all vibrate, snooze off", []byte{0, 3, 3, 3, 3, 3, 0, 0, 16},
			[]string{"02fce4fc2100030303030300001018b66820aab8", "01fc14e23ecd3308bb03f0af5a43ac1f63135fda", "00fc3076"}, -4},
		{"all vibrate, general beep", []byte{0, 3, 0, 3, 3, 3, 0, 0, 4},
			[]string{"023ee43e2100030003030300000466b96820dcfb", "013e51b3f7f5e423ff70485325af14cf9e1581e1", "003e2192"}, 62},
		{"all vibrate, quick bolus beep", []byte{0, 0, 3, 3, 3, 3, 0, 0, 6},
			[]string{"0241e4412100000303030300000678b9682013b8", "01416774d031c6e0e61484fb9afe3f6383ad2dc2", "004181de"}, 65},
		{"all vibrate, reminders beep", []byte{0, 3, 3, 0, 3, 3, 0, 0, 8},
			[]string{"026be46b21000303000303000008f1b9682085de", "016bfbade6e070ae43679f929b46a1107db5f2e9", "006bd07d"}, 107},
		{"all vibrate, alarm beep", []byte{0, 3, 3, 3, 3, 0, 0, 0, 34},
			[]string{"0244e444210003030303000000228db96820c33d", "01448a8d342669baf97ac2f7f83e059482139e68", "0044928b"}, 68},
		{"all vibrate, alert beep", []byte{0, 3, 3, 3, 0, 3, 0, 0, 48},
			[]string{"024ee44e21000303030003000030a5b968201053", "014ed8b377b6d8587d9c4e78b31fc2c309586fa6", "004e7101"}, 78},
		{"all vibrate, CGM alert beep", []byte{0, 3, 3, 3, 3, 3, 0, 2, 0x90},
			[]string{"0251e45121000303030303000290b7b96820bd57", "0151cdff118cfdbb501489e5794509c4e23d722c", "0051c6af"}, 81},
		{"all vibrate, CGM alert hypo repeat", []byte{0, 3, 3, 3, 3, 3, 0, 3, 0x80},
			[]string{"0254e45421000303030303000380c9b96820ca71", "0154f054052110f1cf33eb862cadeaac9f0dc1a6", "0054b049"}, 84},
		{"all vibrate, CGM alert vibrate", []byte{0, 3, 3, 3, 3, 3, 0, 0, 0x80},
			[]string{"0260e46021000303030303000080dab96820345f", "0160a95aac1b491ac7317041b822781728cbfa90", "00601f19"}, 96},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := control.NewSetPumpSoundsRequestFromCargo(tc.cargo)
			if err != nil {
				t.Fatal(err)
			}
			parity.TestMessage(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...)
		})
	}
}

func TestSetPumpSoundsRequestDecodesTheAnnunciations(t *testing.T) {
	m, err := control.NewSetPumpSoundsRequestFromCargo([]byte{0, 3, 0, 3, 3, 9, 0, 3, 0x80})
	if err != nil {
		t.Fatal(err)
	}
	if m.QuickBolusAnnun == nil || *m.QuickBolusAnnun != currentstatus.AnnunciationVibrate ||
		m.GeneralAnnun == nil || *m.GeneralAnnun != currentstatus.AnnunciationAudioHigh || m.AlarmAnnun != nil {
		t.Errorf("parsed %+v", m)
	}
	if m.CgmAlertAnnun == nil || *m.CgmAlertAnnun != control.CgmAlertAnnunciationHypoRepeat {
		t.Errorf("CGM alert %v", m.CgmAlertAnnun)
	}
	if !reflect.DeepEqual(m.ChangeBitmask, []control.ChangeBitmask{}) {
		t.Errorf("change bitmask %v", m.ChangeBitmask)
	}
}

func TestSetPumpSoundsRequestFromAnnunciations(t *testing.T) {
	m := control.NewSetPumpSoundsRequest(currentstatus.AnnunciationVibrate, currentstatus.AnnunciationAudioHigh, currentstatus.AnnunciationVibrate,
		currentstatus.AnnunciationVibrate, currentstatus.AnnunciationVibrate, control.CgmAlertAnnunciationBeep, control.ChangeBitmaskGeneral)
	want := []byte{0, 3, 0, 3, 3, 3, 0, 2, 4}
	if !reflect.DeepEqual(m.Cargo(), want) {
		t.Errorf("cargo %x, want %x", m.Cargo(), want)
	}
	if !reflect.DeepEqual(m.ChangeBitmask, []control.ChangeBitmask{control.ChangeBitmaskGeneral}) {
		t.Errorf("change bitmask %v", m.ChangeBitmask)
	}
}

func TestSetPumpSoundsRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "SetPumpSoundsRequest")
}
