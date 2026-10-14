package currentstatus_test

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLastBGResponseEmpty(t *testing.T) {
	parity.TestMessage(t, "0003330307000000000000003117", 3, 1, messages.CurrentStatus, currentstatus.NewLastBGResponse(0, 0, 0))
}

func TestLastBGResponse(t *testing.T) {
	for _, tc := range []struct {
		name      string
		hex       string
		txID      int
		timestamp int64
		value     int32
		source    currentstatus.BgSource
		at        string
	}{
		{"from the CGM", "000e330e0736a0541bce0001d9e4", 14, 458530870, 206, currentstatus.BgSourceCGM, "2022-07-13T01:41:10Z"},
		{"entered", "00d733d70714975a20a90000cc8f", -41, 542807828, 169, currentstatus.BgSourceManual, "2025-03-14T11:57:08Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := currentstatus.NewLastBGResponse(tc.timestamp, tc.value, tc.source.ID())
			parsed := parity.TestMessage(t, tc.hex, tc.txID, 1, messages.CurrentStatus, expected).(*currentstatus.LastBGResponse)
			want, _ := time.Parse(time.RFC3339, tc.at)
			if at, err := parsed.BgTimestampInstant(); err != nil || !at.Equal(want) {
				t.Errorf("at %v (%v), want %v", at, err, want)
			}
			if expected.BgSource == nil || *expected.BgSource != tc.source {
				t.Errorf("source %v, want %v", expected.BgSource, tc.source)
			}
		})
	}
}

func TestLastBGResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LastBGResponse")
}
