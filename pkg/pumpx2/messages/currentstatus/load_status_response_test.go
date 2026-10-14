package currentstatus_test

import (
	"bytes"
	"testing"

	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/currentstatus"
	"github.com/jwoglom/faketandem/pkg/pumpx2/messages/parity"
)

func TestLoadStatusResponse(t *testing.T) {
	tubing := func(s currentstatus.PrimeTubingStatus) *currentstatus.PrimeTubingStatus { return &s }
	nudge := func(s currentstatus.PrimeNudgeStatus) *currentstatus.PrimeNudgeStatus { return &s }
	for _, tc := range []struct {
		name      string
		raw       []byte
		active    bool
		state     currentstatus.LoadState
		tubing    *currentstatus.PrimeTubingStatus
		nudge     *currentstatus.PrimeNudgeStatus
		isLoading bool
	}{
		{"filling the tubing, cannot exit", []byte{1, 2, 3}, true, currentstatus.LoadStatePrimeTubing,
			tubing(currentstatus.PrimeTubingStatusEnteredCannotExit), nil, true},
		{"filling the tubing", []byte{1, 2, 0}, true, currentstatus.LoadStatePrimeTubing, tubing(currentstatus.PrimeTubingStatusStop), nil, true},
		{"tubing filled", []byte{0, 4, 1}, false, currentstatus.LoadStatePrimeNudge, nil, nudge(currentstatus.PrimeNudgeStatusComplete), false},
		{"filling the cannula", []byte{1, 4, 1}, true, currentstatus.LoadStatePrimeNudge, nil, nudge(currentstatus.PrimeNudgeStatusComplete), false},
		{"cannula filled", []byte{0, 3, 0}, false, currentstatus.LoadStatePrimeCannula, nil, nil, false},
		{"changing the cartridge", []byte{1, 0, 0}, true, currentstatus.LoadStateChangeCartridge, nil, nil, true},
		{"cartridge changed", []byte{1, 1, 0}, true, currentstatus.LoadStateLoadCartridge, nil, nil, true},
		{"invalid", []byte{0, 5, 0}, false, currentstatus.LoadStateInvalid, nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, err := currentstatus.NewLoadStatusResponseFromCargo(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			if m.IsLoadingActiveID != int32(tc.raw[0]) || m.IsLoadingActive != tc.active ||
				m.LoadStateID != int32(tc.raw[1]) || m.LoadState != tc.state || m.PrimeStatusID != int32(tc.raw[2]) {
				t.Errorf("parsed %+v", m)
			}
			if !samePrimeTubingStatus(m.PrimeTubingStatus, tc.tubing) || !samePrimeNudgeStatus(m.PrimeNudgeStatus, tc.nudge) {
				t.Errorf("prime statuses %v, %v, want %v, %v", m.PrimeTubingStatus, m.PrimeNudgeStatus, tc.tubing, tc.nudge)
			}
			if m.IsInLoadingState() != tc.isLoading {
				t.Errorf("in a loading state: %v", m.IsInLoadingState())
			}
		})
	}
}

func TestLoadStatusResponseCargoTakesThePrimeStatusOfItsState(t *testing.T) {
	suspended := currentstatus.PrimeTubingStatusSuspended
	built := currentstatus.LoadStatusResponseCargo(true, currentstatus.LoadStatePrimeTubing, &suspended, nil)
	if !bytes.Equal(built, []byte{1, 2, 4}) {
		t.Fatalf("cargo %v", built)
	}
	m, err := currentstatus.NewLoadStatusResponseFromCargo(built)
	if err != nil {
		t.Fatal(err)
	}
	if m.PrimeTubingStatus == nil || *m.PrimeTubingStatus != suspended || m.PrimeNudgeStatus != nil {
		t.Errorf("prime statuses %v, %v", m.PrimeTubingStatus, m.PrimeNudgeStatus)
	}
}

func TestLoadStatusResponseRefusesTheWrongSize(t *testing.T) {
	if err := (&currentstatus.LoadStatusResponse{}).Parse([]byte{1, 2}); err == nil {
		t.Error("a two-byte payload parsed")
	}
}

func TestLoadStatusResponseMatchesJar(t *testing.T) {
	parity.CheckClass(t, "LoadStatusResponse")
}

func samePrimeTubingStatus(a, b *currentstatus.PrimeTubingStatus) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}

func samePrimeNudgeStatus(a, b *currentstatus.PrimeNudgeStatus) bool {
	return (a == nil) == (b == nil) && (a == nil || *a == *b)
}
