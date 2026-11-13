package state

import (
	"encoding/binary"
	"math"
	"testing"
	"time"
)

func frozenSuspendedPump(t *testing.T) (*PumpState, *ManualClock) {
	t.Helper()
	ps := suspendedPump(t)
	clock := NewFrozenClock(time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC))
	ps.SetClock(clock)
	return ps, clock
}

func lastEntry(t *testing.T, ps *PumpState, name string) HistoryLogEntry {
	t.Helper()
	_, last := ps.GetHistoryLogSequenceRange()
	entries := ps.GetHistoryLogEntries(0, last)
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Type == name {
			return entries[i]
		}
	}
	t.Fatalf("no %s record", name)
	return HistoryLogEntry{}
}

func wantLoadStatus(t *testing.T, ps *PumpState, step string, active bool, loadState, prime int) {
	t.Helper()
	if got := ps.CurrentLoadStatus(); got != (LoadStatus{Active: active, LoadStateID: loadState, PrimeStatusID: prime}) {
		t.Errorf("%s: load status %+v, want active=%v state=%d prime=%d", step, got, active, loadState, prime)
	}
}

func TestLoadStatusFollowsTheCartridgeChange(t *testing.T) {
	ps, _ := frozenSuspendedPump(t)

	wantLoadStatus(t, ps, "idle", false, 0, 0)
	if !ps.EnterChangeCartridge(9) {
		t.Fatal("change cartridge refused")
	}
	wantLoadStatus(t, ps, "change mode", true, LoadStateChangeCartridge, 0)
	if _, err := ps.MarkReadyToChange(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ps.InsertCartridge(250); err != nil {
		t.Fatal(err)
	}
	if accepted, detect := ps.ExitChangeCartridge(); !accepted || !detect {
		t.Fatalf("exit accepted=%v detect=%v, want both", accepted, detect)
	}
	wantLoadStatus(t, ps, "checking", true, LoadStateLoadCartridge, 0)
	if units, ok := ps.FinishCartridgeDetection(); !ok || units != 250 {
		t.Fatalf("check finished with %v, %v", units, ok)
	}
	wantLoadStatus(t, ps, "checked", true, LoadStateLoadCartridge, 0)
}

func TestLoadStatusFollowsTheTubingAndCannulaFills(t *testing.T) {
	ps, clock := frozenSuspendedPump(t)
	ps.SetReservoirLevel(250)
	ps.SetCartridgeLoaded(true)

	if !ps.EnterFillTubing(4) {
		t.Fatal("fill tubing refused")
	}
	wantLoadStatus(t, ps, "fill tubing", true, LoadStatePrimeTubing, PrimeTubingStatusEnteredCanExit)
	if _, err := ps.SetFillTubingButton(true); err != nil {
		t.Fatal(err)
	}
	wantLoadStatus(t, ps, "filling", true, LoadStatePrimeTubing, PrimeTubingStatusStart)
	clock.Advance(10 * time.Second)
	if accepted, units := ps.ExitFillTubing(); !accepted || units != 5 {
		t.Fatalf("exit fill tubing accepted=%v units=%v, want 5 U at 0.5 U/s", accepted, units)
	}
	wantLoadStatus(t, ps, "tubing filled", false, 0, 0)

	if !ps.StartFillCannula() {
		t.Fatal("fill cannula refused")
	}
	wantLoadStatus(t, ps, "cannula", true, LoadStatePrimeCannula, 0)
	if !ps.FinishFillCannula(0.3) {
		t.Fatal("no prime to finish")
	}
	wantLoadStatus(t, ps, "done", false, 0, 0)
	if got := ps.GetReservoirLevel(); math.Abs(got-244.7) > 1e-9 {
		t.Errorf("reservoir %v, want 250 less 5 for the tubing and 0.3 for the cannula", got)
	}
}

func TestCartridgeFilledRecordsWhatWasPutIn(t *testing.T) {
	ps, _ := frozenSuspendedPump(t)
	ps.EnterChangeCartridge(1)
	if _, err := ps.MarkReadyToChange(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ps.InsertCartridge(180.4); err != nil {
		t.Fatal(err)
	}
	ps.ExitChangeCartridge()
	ps.FinishCartridgeDetection()

	record := lastEntry(t, ps, "CartridgeFilled")
	if record.TypeID != HistoryCartridgeFilled || record.Data["insulinDisplay"] != 180 || record.Data["insulinActual"] != 180.4 {
		t.Errorf("CartridgeFilled %d %v", record.TypeID, record.Data)
	}
	payload := record.EncodeRecord()[historyPayloadOffset:]
	if got := math.Float32frombits(uint32(payload[4]) | uint32(payload[5])<<8 | uint32(payload[6])<<16 | uint32(payload[7])<<24); got != float32(180.4) {
		t.Errorf("insulinActual on the wire %v", got)
	}
	if payload[0] != 180 {
		t.Errorf("insulinDisplay on the wire %d", payload[0])
	}
}

func TestExitWithNoCartridgePutInChecksTheOneStillThere(t *testing.T) {
	ps, _ := frozenSuspendedPump(t)
	ps.SetReservoirLevel(42)
	ps.EnterChangeCartridge(1)
	if accepted, detect := ps.ExitChangeCartridge(); !accepted || !detect {
		t.Fatalf("exit accepted=%v detect=%v", accepted, detect)
	}
	if units, _ := ps.FinishCartridgeDetection(); units != 42 {
		t.Errorf("checked %v U, want the 42 U already in", units)
	}
}

func TestANewCartridgeRefusesChangeModeUntilItsTubingIsFilled(t *testing.T) {
	ps, clock := frozenSuspendedPump(t)
	ps.EnterChangeCartridge(1)
	ps.ExitChangeCartridge()
	if ps.EnterChangeCartridge(2) {
		t.Fatal("change cartridge accepted while the pump checks a cartridge")
	}
	ps.FinishCartridgeDetection()
	if ps.EnterChangeCartridge(2) {
		t.Fatal("change cartridge accepted with a new cartridge waiting for its tubing fill")
	}

	ps.EnterFillTubing(3)
	if _, units := ps.ExitFillTubing(); units != 0 {
		t.Fatalf("a fill with no press took %v U", units)
	}
	if ps.EnterChangeCartridge(2) {
		t.Fatal("a fill that took nothing completed the cartridge change")
	}

	ps.EnterFillTubing(3)
	if _, err := ps.SetFillTubingButton(true); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Second)
	ps.ExitFillTubing()
	if !ps.EnterChangeCartridge(2) {
		t.Fatal("change cartridge still refused after the tubing filled")
	}
}

func TestFillTubingCapsAPressAtMaxFillSeconds(t *testing.T) {
	ps, clock := frozenSuspendedPump(t)
	ps.UpdateCartridgeSettings(func(s *CartridgeSettings) {
		s.MaxFillSeconds = 4
		s.FillTubingUnitsPerSecond = 1
	})
	ps.EnterFillTubing(3)
	if _, err := ps.SetFillTubingButton(true); err != nil {
		t.Fatal(err)
	}
	clock.Advance(30 * time.Second)
	if _, err := ps.SetFillTubingButton(false); err != nil {
		t.Fatal(err)
	}
	if _, err := ps.SetFillTubingButton(true); err != nil {
		t.Fatal(err)
	}
	clock.Advance(2 * time.Second)
	_, units := ps.ExitFillTubing()
	if units != 6 {
		t.Errorf("filled %v U, want 4 from the capped press and 2 from the second", units)
	}
	record := lastEntry(t, ps, "TubingFilled")
	if record.Data["primeSize"] != 6.0 || record.Data["completionStatus"] != FillCompletionStatusCompleted {
		t.Errorf("TubingFilled %v", record.Data)
	}
	payload := record.EncodeRecord()[historyPayloadOffset:]
	if got := math.Float32frombits(binary.LittleEndian.Uint32(payload[0:4])); got != 6 || binary.LittleEndian.Uint32(payload[4:8]) != 3 {
		t.Errorf("TubingFilled payload %x, want primeSize 6 and completionStatus 3", payload)
	}
}

func TestFillCannulaNeedsASuspendedPumpWithNothingOpen(t *testing.T) {
	ps := NewPumpState()
	if ps.StartFillCannula() {
		t.Error("cannula fill accepted while delivering")
	}
	ps.SetPumpingSuspended(true)
	ps.SetWorkflowMode(WorkflowFillTubing)
	if ps.StartFillCannula() {
		t.Error("cannula fill accepted in fill tubing mode")
	}
	ps.SetWorkflowMode(WorkflowNone)
	if !ps.StartFillCannula() || ps.StartFillCannula() {
		t.Error("want the first fill accepted and a second refused while it runs")
	}
}
