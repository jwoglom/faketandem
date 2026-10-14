package control_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestDismissNotificationRequest(t *testing.T) {
	for _, tc := range []struct {
		name     string
		cargo    []byte
		txID     int
		packets  []string
		wantID   int64
		wantType control.NotificationType
	}{
		{"site change reminder", []byte{2, 0, 0, 0, 0, 0}, 30,
			[]string{"011eb81e1e02000000000091eef21fb2f51e9b10", "001e1923326c7764e514a7cd6e702210fbe0f0"},
			2, control.NotificationTypeReminder},
		{"CGM graph removed alert", []byte{25, 0, 0, 0, 1, 0}, 0xdd,
			[]string{"01ddb8dd1e1900000001001b92f41f4ebdd42d94", "00dd71575252fcc1476112db4196041031cf27"},
			25, control.NotificationTypeAlert},
		{"invalid transmitter ID alert", []byte{29, 0, 0, 0, 1, 0}, 18,
			[]string{"0112b8121e1d00000001008190fd1fd4f0eef6a2", "0012398a829fd5358e41f168f0b82619ad9976"},
			29, control.NotificationTypeAlert},
		{"G6 sensor failed CGM alert", []byte{11, 0, 0, 0, 3, 0}, 0xe9,
			[]string{"01e9b8e91e0b0000000300de6e392075306d24bb", "00e92094d6d88830447aa4c9d10af3196a91a1"},
			11, control.NotificationTypeCGMAlert},
		{"pump reset alarm", []byte{3, 0, 0, 0, 2, 0}, 0xa6,
			[]string{"01a6b8a61e03000000020058e93920faf75ae477", "00a674346fb57c95916f8fde8e2e79f605dee3"},
			3, control.NotificationTypeAlarm},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := control.NewDismissNotificationRequestFromCargo(tc.cargo)
			if err != nil {
				t.Fatal(err)
			}
			parsed := parity.TestMessage(t, tc.packets[0], tc.txID, 1, messages.Control, expected, tc.packets[1:]...).(*control.DismissNotificationRequest)
			if parsed.NotificationID != tc.wantID {
				t.Errorf("notification ID %d, want %d", parsed.NotificationID, tc.wantID)
			}
			if parsed.NotificationType == nil || *parsed.NotificationType != tc.wantType {
				t.Errorf("notification type %v, want %v", parsed.NotificationType, tc.wantType)
			}
			if built := control.NewDismissNotificationRequest(tc.wantType, tc.wantID); messages.ToString(built) != messages.ToString(parsed) {
				t.Errorf("built %s, parsed %s", messages.ToString(built), messages.ToString(parsed))
			}
		})
	}
}

func TestDismissNotificationRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "DismissNotificationRequest")
}
