package control_test

import (
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/control"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestChangeTimeDateRequest(t *testing.T) {
	at := time.Unix(1739758000, 0) // 2025-02-17T02:06:40Z
	fromCargo, err := control.NewChangeTimeDateRequestFromCargo([]byte{48, 23, 57, 32})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		expected *control.ChangeTimeDateRequest
	}{
		{"from its payload", fromCargo},
		{"from a time", control.NewChangeTimeDateRequestAt(at)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed := parity.TestMessage(t, "0143d6431c30173920965d39207d921e9438912a", 67, 1, messages.Control,
				tc.expected, "0043998f1154a96aa50bda699dfde690a9").(*control.ChangeTimeDateRequest)
			if parsed.TandemEpochTime != 540612400 {
				t.Errorf("pump time %d, want 540612400", parsed.TandemEpochTime)
			}
			if !parsed.Instant.Equal(at) {
				t.Errorf("instant %s, want %s", parsed.Instant, at)
			}
		})
	}
}

func TestChangeTimeDateRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "ChangeTimeDateRequest")
}
