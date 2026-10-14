package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestRemoteCarbEntryRequest(t *testing.T) {
	for _, tc := range []struct {
		name           string
		timeSinceReset string
		carbs          int32
		pumpTime       int64
		bolusID        int32
		txID           int
		packets        []string
	}{
		{"5g for bolus 10677", "461710145", 5, 1200239, 10677, -86,
			[]string{"02aaf2aa210500016f501200b5294123851bf324", "01aaa4f96f10fe37167c6d90d2802fda729c1e0e", "00aa0288"}},
		{"3g for bolus 10678", "461710202", 3, 1200239, 10678, -49,
			[]string{"02cff2cf210300016f501200b6297a23851bc88f", "01cfdcc573801e4a91567fe0219a9275211f7f17", "00cfa162"}},
		{"11g for bolus 10653", "461589420", 11, 1079469, 10653, 56,
			[]string{"0238f238210b0001ad7810009d29ac4b831b5e16", "013890d5f066c547ed4e3e883476f805abc0ddf9", "00383346"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := control.NewRemoteCarbEntryRequestWithUnknown(tc.carbs, 1, tc.pumpTime, tc.bolusID)
			if err != nil {
				t.Fatal(err)
			}
			tester := parity.MessageTester{Env: messages.Env{PairingCode: "6VeDeRAL5DCigGw2", TimeSinceReset: tc.timeSinceReset}}
			tester.Test(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...)
		})
	}
}

func TestRemoteCarbEntryRequestRefusesAUnixTime(t *testing.T) {
	if _, err := control.NewRemoteCarbEntryRequest(10, 1700000000, 1); err == nil {
		t.Error("a Unix time was taken for seconds since boot")
	}
}

func TestRemoteCarbEntryRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "RemoteCarbEntryRequest")
}
