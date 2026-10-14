package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/models"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestRemoteBgEntryRequest(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		pairingCode          string
		timeSinceReset       string
		bg                   int32
		useForCgmCalibration bool
		pumpTime             int64
		bolusID              int32
		txID                 int
		packets              []string
	}{
		{"bolus 10676", "6VeDeRAL5DCigGw2", "461710145", 180, false, 1200173, 10676, 59,
			[]string{"023bb63b23b4000000012d501200b4290023851b", "013b1f62ef2f006c8a8ea7aeb9203fa4b7eaebd2", "003bc0b6800c"}},
		{"bolus 10677", "6VeDeRAL5DCigGw2", "461710145", 185, false, 1200239, 10677, -87,
			[]string{"02a9b6a923b9000000016f501200b5294123851b", "01a93ade55510ac65b57f647a1899c0a6e4e94bc", "00a9d8306d38"}},
		{"bolus 10678", "6VeDeRAL5DCigGw2", "461710202", 186, false, 1200239, 10678, -50,
			[]string{"02ceb6ce23ba000000016f501200b6297a23851b", "01ceb50b6d51ca170c5ecf018b37a14a4a05c412", "00ceda1ff8b1"}},
		{"bolus 10652", "6VeDeRAL5DCigGw2", "461589158", 142, false, 1079274, 10652, 60,
			[]string{"023cb63c238e00000001ea7710009c29bc4a831b", "013c1d2edd239e1a8499a6686078565ad1b8acdc", "003ca71a3107"}},
		{"G7 calibration at 169 mg/dL", "", "0", 169, true, 2887044, 0, -42,
			[]string{"02d6b6d623a900010001840d2c00000052cf5a20", "01d6e4b43557b2ab113cb54266a2d1616c1972f8", "00d66e6d6786"}},
		{"G7 calibration at 170 mg/dL", "", "0", 170, true, 2887044, 0, -31,
			[]string{"02e1b6e123aa00010001840d2c00000082cf5a20", "01e10d94486401b99108897a2c3fdb9c2ec6be15", "00e1663cfd1a"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := control.NewRemoteBgEntryRequest(tc.bg, tc.useForCgmCalibration, models.BloodGlucoseReadingTypeManual,
				models.BloodGlucoseReadingSourceRemote, tc.pumpTime, tc.bolusID)
			tester := parity.MessageTester{Env: messages.Env{PairingCode: tc.pairingCode, TimeSinceReset: tc.timeSinceReset}}
			parsed := tester.Test(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...).(*control.RemoteBgEntryRequest)
			if !parsed.IsAutopopBG() {
				t.Error("a REMOTE reading is not autopopulated")
			}
		})
	}
}

func TestManualRemoteBgEntryRequestTakesItsSourceFromIsAutopopBg(t *testing.T) {
	for _, isAutopopBg := range []bool{false, true} {
		m := control.NewManualRemoteBgEntryRequest(120, false, isAutopopBg, 1000, 1)
		if m.EntryType != models.BloodGlucoseReadingTypeManual || m.IsAutopopBG() != isAutopopBg {
			t.Errorf("isAutopopBg %v: entry type %v, source %v", isAutopopBg, m.EntryType, m.Source)
		}
	}
}

func TestRemoteBgEntryRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "RemoteBgEntryRequest")
}
