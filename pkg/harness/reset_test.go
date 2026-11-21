package harness

import (
	"net/http"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/state"
)

const secondsPerDay = 86400

func pumpLog(ps *state.PumpState) []state.HistoryLogEntry {
	first, last := ps.GetHistoryLogSequenceRange()
	return ps.GetHistoryLogEntries(first, last)
}

// entriesOfType returns the entries named name, in log order.
func entriesOfType(entries []state.HistoryLogEntry, name string) []state.HistoryLogEntry {
	var found []state.HistoryLogEntry
	for _, e := range entries {
		if e.Type == name {
			found = append(found, e)
		}
	}
	return found
}

func onlyEntry(t *testing.T, entries []state.HistoryLogEntry, name string) state.HistoryLogEntry {
	t.Helper()
	found := entriesOfType(entries, name)
	if len(found) != 1 {
		t.Fatalf("%d %s records; want 1", len(found), name)
	}
	return found[0]
}

func TestOutOfShelfModeLeavesTheOldBolusesBehindAShelfModeRecordAndTheClockOn2020(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	if _, err := h.ApplyScenario(Scenario{Preset: "out-of-shelf-mode"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	shelf := onlyEntry(t, entries, "ShelfMode")
	newDay := entriesOfType(entries, "NewDay")
	armInit := onlyEntry(t, entries, "ArmInit")
	version := onlyEntry(t, entries, "VersionInfo")
	dateChange := onlyEntry(t, entries, "DateChange")
	timeChanged := onlyEntry(t, entries, "TimeChanged")
	if len(newDay) == 0 || newDay[len(newDay)-1].Sequence < shelf.Sequence {
		t.Fatalf("no NewDay record after the ShelfMode record at %d", shelf.Sequence)
	}
	reset := newDay[len(newDay)-1]
	order := []state.HistoryLogEntry{shelf, reset, armInit, version, dateChange, timeChanged}
	for i := 1; i < len(order); i++ {
		if order[i].Sequence <= order[i-1].Sequence {
			t.Errorf("%s at %d comes before %s at %d", order[i].Type, order[i].Sequence, order[i-1].Type, order[i-1].Sequence)
		}
	}
	for _, bolus := range entriesOfType(entries, "BolusCompleted") {
		if bolus.Sequence > shelf.Sequence {
			t.Errorf("a bolus after the ShelfMode record: %+v", bolus)
		}
		if bolus.PumpTime <= armInit.PumpTime {
			t.Errorf("a bolus before the reset stamped %d, not after the restarted clock's %d", bolus.PumpTime, armInit.PumpTime)
		}
	}
	if armInit.PumpTime != resetPumpSeconds {
		t.Errorf("ArmInit stamped %d; want the restarted clock's %d", armInit.PumpTime, resetPumpSeconds)
	}
	if got := toInt(dateChange.Data["dateAfter"]) - toInt(dateChange.Data["datePrior"]); got != 4383 {
		t.Errorf("DateChange moves the date %d days; want 4383 (to 2020-01-01)", got)
	}
	// Set to 2020-01-01 00:00:01 four minutes after waking, 436 minutes ago.
	if got, want := ps.PumpTimeNow(), uint32(4383*secondsPerDay+1+436*60); got != want {
		t.Errorf("the pump's clock reads %d; want %d (2020-01-01 07:16:01)", got, want)
	}
	if got := ps.GetTimeSinceReset(); got != 440*60 {
		t.Errorf("time since reset %d s; want %d", got, 440*60)
	}
	if !ps.IsPumpingSuspended() {
		t.Error("the pump delivers; it has been suspended since it was put away")
	}
}

func TestAPumpResetRestartsItsClockWithNoChangeRecord(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	if _, err := h.ApplyScenario(Scenario{Preset: "pump-reset"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	if n := len(entriesOfType(entries, "TimeChanged")) + len(entriesOfType(entries, "DateChange")); n != 0 {
		t.Errorf("%d clock change records; a reset writes none", n)
	}
	if n := len(entriesOfType(entries, "ShelfMode")); n != 0 {
		t.Errorf("%d ShelfMode records; want none", n)
	}
	armInit := onlyEntry(t, entries, "ArmInit")
	if armInit.PumpTime != resetPumpSeconds || !armInit.Timestamp.Equal(testInstant.Add(-180*time.Minute)) {
		t.Errorf("ArmInit at %s stamped %d; want 3 hours ago, stamped %d", armInit.Timestamp, armInit.PumpTime, resetPumpSeconds)
	}
	if got, want := ps.PumpTimeNow(), uint32(resetPumpSeconds+180*60); got != want {
		t.Errorf("the pump's clock reads %d; want %d, three hours on from 2008-01-01", got, want)
	}
	boluses := entriesOfType(entries, "BolusActivated")
	last := boluses[len(boluses)-1]
	if last.Sequence < armInit.Sequence || last.PumpTime != resetPumpSeconds+120*60 {
		t.Errorf("the bolus after the reset at %d stamped %d; want after %d, stamped on the restarted clock", last.Sequence, last.PumpTime, armInit.Sequence)
	}
}

func TestARebootThatKeepsTheClockMovesNoStamp(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	before := ps.PumpTimeNow()
	if _, err := h.ApplyScenario(Scenario{Preset: "reboot-kept-clock"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	if n := len(entriesOfType(entries, "NewDay")); n != 0 {
		t.Errorf("%d NewDay records; a reboot that keeps the clock writes none", n)
	}
	armInit := onlyEntry(t, entries, "ArmInit")
	if armInit.PumpTime != ps.PumpTimeFor(armInit.Timestamp) {
		t.Errorf("ArmInit stamped %d; want %d, on the clock it kept", armInit.PumpTime, ps.PumpTimeFor(armInit.Timestamp))
	}
	if after := ps.PumpTimeNow(); after != before {
		t.Errorf("the pump's clock reads %d; want %d", after, before)
	}
}

func TestClockSetBackStampsTheBolusGivenMeanwhileAnHourEarly(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	before := ps.PumpTimeNow()
	if _, err := h.ApplyScenario(Scenario{Preset: "clock-set-back"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	if n := len(entriesOfType(entries, "TimeChanged")); n != 2 {
		t.Errorf("%d TimeChanged records; want 2", n)
	}
	boluses := entriesOfType(entries, "BolusActivated")
	if len(boluses) != 3 {
		t.Fatalf("%d boluses; want 3", len(boluses))
	}
	for i, offset := range []int64{0, -3600, 0} {
		want := int64(before) - int64(testInstant.Sub(boluses[i].Timestamp).Seconds()) + offset
		if int64(boluses[i].PumpTime) != want {
			t.Errorf("bolus %d stamped %d; want %d", i+1, boluses[i].PumpTime, want)
		}
	}
	if after := ps.PumpTimeNow(); after != before {
		t.Errorf("the pump's clock reads %d; want %d, set back and forward again", after, before)
	}
}

// stampOnTheClockNow is e's stamp moved by every TimeChanged and DateChange
// record after it, as a reader of the log dates a record.
func stampOnTheClockNow(entries []state.HistoryLogEntry, e state.HistoryLogEntry) int64 {
	stamp := int64(e.PumpTime)
	for _, change := range entries {
		if change.Sequence <= e.Sequence {
			continue
		}
		switch change.Type {
		case "TimeChanged":
			stamp += (toInt(change.Data["timeAfter"]) - toInt(change.Data["timePrior"])) / 1000
		case "DateChange":
			stamp += (toInt(change.Data["dateAfter"]) - toInt(change.Data["datePrior"])) * secondsPerDay
		}
	}
	return stamp
}

// checkBolusesDatedByTheLog checks that the clock records date each bolus
// activation to the instant it was given, or, where datable is false, do not.
func checkBolusesDatedByTheLog(t *testing.T, ps *state.PumpState, datable ...bool) []state.HistoryLogEntry {
	t.Helper()
	entries := pumpLog(ps)
	boluses := entriesOfType(entries, "BolusActivated")
	if len(boluses) != len(datable) {
		t.Fatalf("%d boluses; want %d", len(boluses), len(datable))
	}
	for i, b := range boluses {
		dated, given := stampOnTheClockNow(entries, b), int64(ps.PumpTimeFor(b.Timestamp))
		if (dated == given) != datable[i] {
			t.Errorf("bolus %d given at %d on the clock now is dated %d by the log; datable %v", i+1, given, dated, datable[i])
		}
	}
	return boluses
}

func countClockChanges(entries []state.HistoryLogEntry) int {
	return len(entriesOfType(entries, "TimeChanged")) + len(entriesOfType(entries, "DateChange"))
}

func TestClockDriftedAheadStampsEveryRecordSevenMinutesAheadWithNoRecordOfIt(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	before := ps.PumpTimeNow()
	if _, err := h.ApplyScenario(Scenario{Preset: "clock-drifted-ahead"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	if n := countClockChanges(entries); n != 0 {
		t.Errorf("%d clock change records; a drift writes none", n)
	}
	if got, want := ps.PumpTimeNow(), before+420; got != want {
		t.Errorf("the pump's clock reads %d; want %d", got, want)
	}
	for _, e := range entries {
		if want := ps.PumpTimeFor(e.Timestamp); e.PumpTime != want {
			t.Fatalf("%s at %d stamped %d; want %d, on the clock 7 minutes ahead", e.Type, e.Sequence, e.PumpTime, want)
		}
	}
	checkBolusesDatedByTheLog(t, ps, true, true, true)
}

func TestClockNudgedBackStampsTheLaterBolusBeforeTheEarlierOne(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	before := ps.PumpTimeNow()
	if _, err := h.ApplyScenario(Scenario{Preset: "clock-nudged-back"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	changed := onlyEntry(t, entries, "TimeChanged")
	if got := toInt(changed.Data["timeAfter"]) - toInt(changed.Data["timePrior"]); got != -180000 {
		t.Errorf("TimeChanged moves the clock %d ms; want -180000", got)
	}
	if n := len(entriesOfType(entries, "DateChange")); n != 0 {
		t.Errorf("%d DateChange records; want none", n)
	}
	boluses := checkBolusesDatedByTheLog(t, ps, true, true, true)
	if boluses[2].PumpTime >= boluses[1].PumpTime {
		t.Errorf("the bolus after the change stamped %d, not before the one ahead of it at %d", boluses[2].PumpTime, boluses[1].PumpTime)
	}
	if got := ps.PumpTimeNow(); got != before {
		t.Errorf("the pump's clock reads %d; want %d, set right again", got, before)
	}
}

func TestClockSetAheadHoursLeavesThePumpSixHoursAheadWithEveryBolusDatable(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	before := ps.PumpTimeNow()
	if _, err := h.ApplyScenario(Scenario{Preset: "clock-set-ahead-hours"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	if n := len(entriesOfType(entries, "TimeChanged")); n != 1 {
		t.Errorf("%d TimeChanged records; want 1", n)
	}
	boluses := checkBolusesDatedByTheLog(t, ps, true, true, true)
	for i, offset := range []int64{0, 21600, 21600} {
		want := int64(before) - int64(testInstant.Sub(boluses[i].Timestamp).Seconds()) + offset
		if int64(boluses[i].PumpTime) != want {
			t.Errorf("bolus %d stamped %d; want %d", i+1, boluses[i].PumpTime, want)
		}
	}
	if got, want := ps.PumpTimeNow(), before+21600; got != want {
		t.Errorf("the pump's clock reads %d; want %d", got, want)
	}
}

func TestClockDateSetBackWritesOnlyADateChangeAndStampsTheLastBolusAMonthEarly(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	before := ps.PumpTimeNow()
	if _, err := h.ApplyScenario(Scenario{Preset: "clock-date-set-back"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	dateChange := onlyEntry(t, entries, "DateChange")
	if got := toInt(dateChange.Data["dateAfter"]) - toInt(dateChange.Data["datePrior"]); got != -31 {
		t.Errorf("DateChange moves the date %d days; want -31", got)
	}
	if n := len(entriesOfType(entries, "TimeChanged")); n != 0 {
		t.Errorf("%d TimeChanged records; the time of day did not move", n)
	}
	boluses := checkBolusesDatedByTheLog(t, ps, true, true, true)
	if boluses[2].PumpTime >= boluses[0].PumpTime {
		t.Errorf("the bolus after the change stamped %d, not before the first at %d", boluses[2].PumpTime, boluses[0].PumpTime)
	}
	if got, want := ps.PumpTimeNow(), before-31*secondsPerDay; got != want {
		t.Errorf("the pump's clock reads %d; want %d", got, want)
	}
}

func TestClockJumpedBackLeavesEveryEarlierRecordUndatable(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	before := ps.PumpTimeNow()
	if _, err := h.ApplyScenario(Scenario{Preset: "clock-jumped-back"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	if n := countClockChanges(entries); n != 0 {
		t.Errorf("%d clock change records; the jump writes none", n)
	}
	for _, name := range []string{"ArmInit", "NewDay", "ShelfMode"} {
		if n := len(entriesOfType(entries, name)); n != 0 {
			t.Errorf("%d %s records; want none", n, name)
		}
	}
	boluses := checkBolusesDatedByTheLog(t, ps, false, false, true)
	if boluses[2].PumpTime >= boluses[0].PumpTime {
		t.Errorf("the bolus after the jump stamped %d, not before the first at %d", boluses[2].PumpTime, boluses[0].PumpTime)
	}
	if got, want := ps.PumpTimeNow(), before-2*secondsPerDay; got != want {
		t.Errorf("the pump's clock reads %d; want %d", got, want)
	}
}

func TestClockChangePresetsDateEveryRecordByTheLogAndEndWhereTheySay(t *testing.T) {
	for name, endsMinutes := range map[string]int64{
		"clock-fall-back":       0,
		"clock-spring-forward":  0,
		"clock-travel-east":     0,
		"clock-travel-west":     0,
		"clock-travel-12h-east": 0,
		"clock-travel-12h-west": 0,
		"clock-73h-ahead":       73 * 60,
	} {
		t.Run(name, func(t *testing.T) {
			h, ps, _ := scenarioHarness(t)
			before := ps.PumpTimeNow()
			if _, err := h.ApplyScenario(Scenario{Preset: name}); err != nil {
				t.Fatalf("ApplyScenario: %v", err)
			}
			entries := pumpLog(ps)

			if n := len(entriesOfType(entries, "TimeChanged")) + len(entriesOfType(entries, "DateChange")); n == 0 {
				t.Error("no clock change record")
			}
			for _, e := range entries {
				if e.Type == "TimeChanged" || e.Type == "DateChange" {
					continue // a change run is stamped on its final clock
				}
				if dated, want := stampOnTheClockNow(entries, e), int64(ps.PumpTimeFor(e.Timestamp)); dated != want {
					t.Errorf("%s at %d is dated %d by the log; written at %d on the clock now", e.Type, e.Sequence, dated, want)
				}
			}
			activated, completed := onlyEntry(t, entries, "TempRateActivated"), onlyEntry(t, entries, "TempRateCompleted")
			if change := entriesOfType(entries, "TimeChanged"); len(change) > 0 &&
				(activated.Sequence >= change[0].Sequence || change[0].Sequence >= completed.Sequence) {
				t.Errorf("the temp rate (%d to %d) does not run across the change at %d", activated.Sequence, completed.Sequence, change[0].Sequence)
			}
			if got, want := int64(ps.PumpTimeNow()), int64(before)+endsMinutes*60; got != want {
				t.Errorf("the pump's clock reads %d; want %d", got, want)
			}
		})
	}
}

func TestClockJumpedAheadLeavesTheBolusBeforeItOnAClockNothingExplains(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	before := ps.PumpTimeNow()
	if _, err := h.ApplyScenario(Scenario{Preset: "clock-jumped-ahead"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}
	entries := pumpLog(ps)

	if n := countClockChanges(entries); n != 0 {
		t.Errorf("%d clock change records; the jump writes none", n)
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].PumpTime < entries[i-1].PumpTime {
			t.Fatalf("%s at %d stamped %d runs back from %d", entries[i].Type, entries[i].Sequence, entries[i].PumpTime, entries[i-1].PumpTime)
		}
	}
	checkBolusesDatedByTheLog(t, ps, false, true)
	if got, want := int64(ps.PumpTimeNow()), int64(before)+(9*365+2)*secondsPerDay; got != want {
		t.Errorf("the pump's clock reads %d; want %d", got, want)
	}
}

func TestAScenarioRefusesAResetWhileThePumpDelivers(t *testing.T) {
	_, _, mux := scenarioHarness(t)

	status, body := do(t, mux, http.MethodPost, "/api/state/scenario", `{"events":[{"minutes_ago":30,"type":"reset"}]}`)
	if status != http.StatusBadRequest {
		t.Fatalf("status %d (%v); want 400", status, body)
	}
}

func TestAClockSetNeedsADelta(t *testing.T) {
	_, _, mux := scenarioHarness(t)

	status, body := do(t, mux, http.MethodPost, "/api/state/scenario", `{"events":[{"minutes_ago":30,"type":"clock_set"}]}`)
	if status != http.StatusBadRequest {
		t.Fatalf("status %d (%v); want 400", status, body)
	}
}

func TestTheResetActionPutsASuspendedPumpOnTheShelfAndRestartsItsClock(t *testing.T) {
	_, ps, mux := scenarioHarness(t)

	if status, body := do(t, mux, http.MethodPost, "/api/state/reset", ""); status != http.StatusConflict {
		t.Fatalf("reset while delivering: status %d (%v); want 409", status, body)
	}
	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"user"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/reset", `{"shelf_mode":true}`)

	entries := pumpLog(ps)
	names := make([]string, 0, 4)
	for _, e := range entries[len(entries)-4:] {
		names = append(names, e.Type)
	}
	if want := []string{"ShelfMode", "NewDay", "ArmInit", "VersionInfo"}; !equalStrings(names, want) {
		t.Errorf("last records %v; want %v", names, want)
	}
	if got := ps.PumpTimeNow(); got != resetPumpSeconds {
		t.Errorf("the pump's clock reads %d; want %d", got, resetPumpSeconds)
	}
	if got := ps.GetTimeSinceReset(); got != 0 {
		t.Errorf("time since reset %d; want 0", got)
	}
}

func TestTheResetActionCanKeepTheClock(t *testing.T) {
	_, ps, mux := scenarioHarness(t)
	before := ps.PumpTimeNow()

	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"user"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/reset", `{"keep_clock":true}`)

	if got := ps.PumpTimeNow(); got != before {
		t.Errorf("the pump's clock reads %d; want %d", got, before)
	}
	if n := len(entriesOfType(pumpLog(ps), "NewDay")); n != 0 {
		t.Errorf("%d NewDay records; want none", n)
	}
}

func toInt(v interface{}) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case uint32:
		return int64(n)
	case float64:
		return int64(n)
	}
	return 0
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
