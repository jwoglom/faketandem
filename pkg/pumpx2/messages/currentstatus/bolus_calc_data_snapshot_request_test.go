package currentstatus_test

import (
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestBolusCalcDataSnapshotRequest(t *testing.T) {
	parity.TestMessage(t, "0003720300a72f", 3, 1, messages.CurrentStatus, currentstatus.NewBolusCalcDataSnapshotRequest())
}

func TestBolusCalcDataSnapshotRequestMatchesJar(t *testing.T) {
	parity.CheckClass(t, "BolusCalcDataSnapshotRequest")
}
