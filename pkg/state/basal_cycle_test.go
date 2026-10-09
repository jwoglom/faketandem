package state

import (
	"math"
	"testing"
	"time"
)

// cyclePump is a pump at 1 U/hr on a frozen clock in UTC, with the basal cycle
// on from t0, so its first 279 is at t0.
func cyclePump(t *testing.T) (*PumpState, *ManualClock, *Simulator, time.Time) {
	t.Helper()
	ps, clock := lifecyclePump(t)
	ps.SetPumpTimeZone(time.UTC)
	ps.SetBasalState(&BasalState{CurrentRate: 1.0})
	t0 := ps.Now()
	ps.SetBasalCycleEnabled(true)
	return ps, clock, NewSimulator(ps, time.Second), t0
}

// advance moves the clock and runs one tick, however long the step.
func advance(clock *ManualClock, sim *Simulator, d time.Duration) {
	clock.Advance(d)
	sim.Tick()
}

type deliveryRecord struct {
	at       time.Time
	sequence uint32
	source   int
	rate     int
	profile  int
}

func records(ps *PumpState, typeName string) []HistoryLogEntry {
	_, last := ps.GetHistoryLogSequenceRange()
	var found []HistoryLogEntry
	for _, entry := range ps.GetHistoryLogEntries(0, last) {
		if entry.Type == typeName {
			found = append(found, entry)
		}
	}
	return found
}

func deliveries(ps *PumpState) []deliveryRecord {
	var found []deliveryRecord
	for _, entry := range records(ps, "BasalDelivery") {
		found = append(found, deliveryRecord{
			at:       entry.Timestamp,
			sequence: entry.Sequence,
			source:   entry.Data["commandedRateSource"].(int),
			rate:     entry.Data["commandedRate"].(int),
			profile:  entry.Data["profileBasalRate"].(int),
		})
	}
	return found
}

func startTemp(ps *PumpState, id, percent int, duration time.Duration) {
	start := ps.Now()
	profile := ps.GetProfileBasalRate()
	ps.EndTempRate(start)
	ps.SetBasalState(&BasalState{
		CurrentRate:      profile,
		TempBasalActive:  true,
		TempBasalRate:    profile * float64(percent) / 100,
		TempBasalPercent: percent,
		TempBasalStart:   start,
		TempBasalEnd:     start.Add(duration),
		TempRateID:       id,
	})
}

func pcmChanges(ps *PumpState) [][2]int {
	var changes [][2]int
	for _, entry := range records(ps, "ControlIQPcmChange") {
		changes = append(changes, [2]int{entry.Data["previousPcm"].(int), entry.Data["currentPcm"].(int)})
	}
	return changes
}

func TestBasalCycleIsOffByDefault(t *testing.T) {
	ps, clock := lifecyclePump(t)
	ps.SetBasalState(&BasalState{CurrentRate: 1.0})
	sim := NewSimulator(ps, time.Second)
	sim.Tick()
	before := ps.GetReservoirLevel()

	advance(clock, sim, time.Hour)

	if n := countRecords(ps, "BasalDelivery"); n != 0 {
		t.Errorf("%d BasalDelivery records with the cycle off, want none", n)
	}
	if delivered := before - ps.GetReservoirLevel(); math.Abs(delivered-1.0) > 1e-6 {
		t.Errorf("delivered %v U in an hour at 1 U/hr, want 1 U continuously", delivered)
	}
}

func TestBasalCycleWritesEach279AtItsBoundaryAcrossOneTick(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	advance(clock, sim, time.Minute)
	startTemp(ps, 3, 150, 12*time.Minute)

	advance(clock, sim, 29*time.Minute)

	got := deliveries(ps)
	want := []struct {
		minute, source, rate int
	}{
		{0, BasalSourceProfile, 1000},
		{5, BasalSourceTemp, 1500},
		{10, BasalSourceTemp, 1500},
		{15, BasalSourceProfile, 1000},
		{20, BasalSourceProfile, 1000},
		{25, BasalSourceProfile, 1000},
		{30, BasalSourceProfile, 1000},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d 279s, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if !got[i].at.Equal(t0.Add(time.Duration(w.minute)*time.Minute)) || got[i].source != w.source || got[i].rate != w.rate {
			t.Errorf("279 %d = %+v, want minute %d source %d rate %d", i, got[i], w.minute, w.source, w.rate)
		}
	}

	completed := records(ps, "TempRateCompleted")
	if len(completed) != 1 || !completed[0].Timestamp.Equal(t0.Add(13*time.Minute)) {
		t.Fatalf("TempRateCompleted = %+v, want one at minute 13", completed)
	}
	if seq := completed[0].Sequence; seq < got[2].sequence || seq > got[3].sequence {
		t.Errorf("the temp rate's end (#%d) is not between the 279s either side of it (#%d, #%d)", seq, got[2].sequence, got[3].sequence)
	}
}

func TestATempSetAndReplacedWithinACycleIsNeverDelivered(t *testing.T) {
	ps, clock, sim, _ := cyclePump(t)
	before := ps.GetReservoirLevel()
	advance(clock, sim, time.Minute)
	startTemp(ps, 1, 200, 30*time.Minute)
	advance(clock, sim, time.Minute)
	startTemp(ps, 2, 50, 30*time.Minute)

	advance(clock, sim, 3*time.Minute)

	got := deliveries(ps)
	if len(got) != 2 || got[1].source != BasalSourceTemp || got[1].rate != 500 {
		t.Fatalf("279s = %+v, want the profile at t0 and the 50%% temp at minute 5", got)
	}
	// The t0 cycle went in when the cycle was turned on, before `before`.
	want := 0.5 * 5 / 60
	if delivered := before - ps.GetReservoirLevel(); math.Abs(delivered-want) > 1e-9 {
		t.Errorf("delivered %v U, want %v: the 200%% temp never ran", delivered, want)
	}
}

func TestATempRateIsExactlyItsPercentOfTheProfile(t *testing.T) {
	ps, clock, sim, _ := cyclePump(t)
	ps.SetBasalState(&BasalState{CurrentRate: 0.85})
	startTemp(ps, 1, 125, 30*time.Minute)

	advance(clock, sim, 5*time.Minute)

	if got := deliveries(ps); got[len(got)-1].rate != 1063 || got[len(got)-1].profile != 850 {
		t.Errorf("279 = %+v, want 125%% of 850 mU/hr, 1062.5 rounded", got[len(got)-1])
	}
}

func TestDailyBasalTotalsTheCyclesIncludingOneASuspendCutShort(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	advance(clock, sim, 7*time.Minute)
	ps.SuspendDelivery("user")
	advance(clock, sim, 9*time.Minute)
	ps.ResumeDelivery()
	advance(clock, sim, 9*time.Minute)

	got := deliveries(ps)
	wantMinutes := []int{0, 5, 10, 15, 16, 21}
	if len(got) != len(wantMinutes) {
		t.Fatalf("279s = %+v, want minutes %v", got, wantMinutes)
	}
	for i, minute := range wantMinutes {
		if !got[i].at.Equal(t0.Add(time.Duration(minute) * time.Minute)) {
			t.Errorf("279 %d at %v, want minute %d", i, got[i].at, minute)
		}
	}
	if got[2].source != BasalSourceSuspended || got[3].source != BasalSourceSuspended || got[4].source != BasalSourceProfile {
		t.Errorf("sources %d %d %d, want suspended, suspended, then the profile at the resume", got[2].source, got[3].source, got[4].source)
	}

	daily := records(ps, "DailyBasal")
	if len(daily) != len(wantMinutes) {
		t.Fatalf("%d DailyBasal records, want one 90 s after each 279", len(daily))
	}
	for i, record := range daily {
		if !record.Timestamp.Equal(got[i].at.Add(90 * time.Second)) {
			t.Errorf("DailyBasal %d at %v, want 90 s after its 279 at %v", i, record.Timestamp, got[i].at)
		}
	}
	// The cycle at minute 5 counts in full though the suspend cut it short.
	last := daily[len(daily)-1].Data["dailyTotalBasal"].(float64)
	if want := 4.0 / 12; math.Abs(last-want) > 1e-9 {
		t.Errorf("dailyTotalBasal = %v, want %v from four cycles at 1 U/hr", last, want)
	}
	if rate := daily[len(daily)-1].Data["lastBasalRate"].(float64); rate != 1.0 {
		t.Errorf("lastBasalRate = %v, want the latest 279's 1 U/hr", rate)
	}
}

func TestAResumeWithinTheSuspendsOwnCycleWritesNo279(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	advance(clock, sim, time.Minute)
	ps.SuspendDelivery("user")
	advance(clock, sim, 2*time.Minute)
	ps.ResumeDelivery()

	advance(clock, sim, 2*time.Minute)

	got := deliveries(ps)
	if len(got) != 2 || !got[1].at.Equal(t0.Add(5*time.Minute)) || got[1].source != BasalSourceProfile {
		t.Errorf("279s = %+v, want t0 and minute 5 with no 279 at the resume", got)
	}
}

func TestControlIQRunsItsRateAndWritesEachPCMChange(t *testing.T) {
	ps, clock, sim, _ := cyclePump(t)
	ps.SetControlIQAlgorithmRate(2.2)
	ps.SetClosedLoopEnabled(true)
	advance(clock, sim, 5*time.Minute)
	ps.SetCGMAvailable(false)
	advance(clock, sim, 5*time.Minute)
	ps.SetCGMAvailable(true)
	advance(clock, sim, 5*time.Minute)
	ps.SuspendDelivery("user")
	ps.ResumeDelivery()
	ps.SetClosedLoopEnabled(false)

	want := [][2]int{
		{PCMOpenLoop, PCMClosedLoop},
		{PCMClosedLoop, PCMNoCGM},
		{PCMNoCGM, PCMClosedLoop},
		{PCMClosedLoop, PCMNoControl},
		{PCMNoControl, PCMClosedLoop},
		{PCMClosedLoop, PCMOpenLoop},
	}
	if got := pcmChanges(ps); len(got) != len(want) {
		t.Fatalf("PCM changes = %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("PCM change %d = %v, want %v", i, got[i], want[i])
			}
		}
	}

	got := deliveries(ps)
	sources := []int{}
	for _, d := range got[1:] {
		sources = append(sources, d.source*10000+d.rate)
	}
	wantSources := []int{
		BasalSourceControlIQ*10000 + 2200,
		BasalSourceProfile*10000 + 1000, // no CGM: the profile
		BasalSourceControlIQ*10000 + 2200,
	}
	if len(sources) != len(wantSources) {
		t.Fatalf("279s after t0 = %+v", got[1:])
	}
	for i := range wantSources {
		if sources[i] != wantSources[i] {
			t.Errorf("279 %d = source %d rate %d, want %d", i+1, sources[i]/10000, sources[i]%10000, wantSources[i])
		}
	}

	// The PCM change for the lost CGM is at the cycle it applies to, before its 279.
	pcms := records(ps, "ControlIQPcmChange")
	if !pcms[1].Timestamp.Equal(got[2].at) || pcms[1].Sequence > got[2].sequence {
		t.Errorf("CGM loss PCM change %+v, want it at and before the 279 %+v", pcms[1], got[2])
	}
}

func TestTurningControlIQOnEndsARunningTemp(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	startTemp(ps, 4, 150, 30*time.Minute)
	advance(clock, sim, 2*time.Minute)

	ps.SetClosedLoopEnabled(true)

	completed := records(ps, "TempRateCompleted")
	if len(completed) != 1 || !completed[0].Timestamp.Equal(t0.Add(2*time.Minute)) {
		t.Errorf("TempRateCompleted = %+v, want one when Control-IQ took over", completed)
	}
}

func TestAnAlarmSuspendKeepingTheTempRunsItAgainAfterTheResume(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	startTemp(ps, 8, 150, time.Hour)
	advance(clock, sim, 7*time.Minute)

	ps.SuspendDeliveryKeepingTemp("alarm")
	advance(clock, sim, 5*time.Minute)
	ps.ResumeDelivery()
	advance(clock, sim, time.Minute)

	if n := countRecords(ps, "TempRateCompleted"); n != 0 {
		t.Errorf("%d TempRateCompleted records, want none: the alarm kept the temp", n)
	}
	got := deliveries(ps)
	last := got[len(got)-1]
	if !last.at.Equal(t0.Add(12*time.Minute)) || last.source != BasalSourceTemp || last.rate != 1500 {
		t.Errorf("279 at the resume = %+v, want the temp again at minute 12", last)
	}
	if got[len(got)-2].source != BasalSourceSuspended {
		t.Errorf("279 at minute 10 = %+v, want suspended", got[len(got)-2])
	}

	advance(clock, sim, time.Hour)
	if completed := records(ps, "TempRateCompleted"); len(completed) != 1 || !completed[0].Timestamp.Equal(t0.Add(time.Hour)) {
		t.Errorf("TempRateCompleted = %+v, want one at the temp's programmed end", completed)
	}
}

func TestATempIsRescaledAtAProfileSegmentBoundary(t *testing.T) {
	ps, clock, sim, t0 := cyclePump(t)
	ps.SetBasalCycleEnabled(false)
	// t0 is 12:00 pump time; the second segment starts at 12:10.
	ps.SetIDPProfiles([]IDPProfile{{ID: 1, Segments: []IDPSegment{
		{StartTime: 0, BasalRate: 1000},
		{StartTime: 12*60 + 10, BasalRate: 2000},
	}}})
	ps.SetBasalCycleEnabled(true)
	startTemp(ps, 2, 150, time.Hour)

	advance(clock, sim, 15*time.Minute)

	got := deliveries(ps)
	var at10 deliveryRecord
	for _, d := range got {
		if d.at.Equal(t0.Add(10 * time.Minute)) {
			at10 = d
		}
	}
	if at10.rate != 3000 || at10.profile != 2000 {
		t.Errorf("279 at 12:10 = %+v, want 150%% of the new segment's 2 U/hr", at10)
	}
	if temp := ps.GetTempRate(); temp.Rate != 3.0 {
		t.Errorf("temp rate = %v U/hr, want it rescaled to 3", temp.Rate)
	}
}

func TestAStagedRateHoldsUntilTheNextSegmentBoundary(t *testing.T) {
	// The default profile runs 0.85 U/hr; the pump was staged at 1 U/hr.
	ps, clock, sim, _ := cyclePump(t)

	advance(clock, sim, 10*time.Minute)

	if got := deliveries(ps); got[len(got)-1].profile != 1000 {
		t.Errorf("279 = %+v, want the staged 1 U/hr, not the profile's", got[len(got)-1])
	}
}

func TestASetClockWritesNo279sAcrossTheJump(t *testing.T) {
	ps, clock, sim, _ := cyclePump(t)
	advance(clock, sim, 2*time.Minute)

	clock.Set(ps.Now().Add(24 * time.Hour))
	sim.Rebase()
	jumpedTo := ps.Now()
	advance(clock, sim, 3*time.Minute)

	got := deliveries(ps)
	if len(got) != 2 || !got[1].at.Equal(jumpedTo.Add(3*time.Minute)) {
		t.Errorf("279s = %+v, want the cadence kept: one 3 minutes after the jump", got)
	}
}

func TestSetTempRateResponseIsWrittenOnlyWithTheCycle(t *testing.T) {
	ps, _ := lifecyclePump(t)
	ps.RecordSetTempRateResponse(0, 5)
	if n := countRecords(ps, "SetTempRateResponse"); n != 0 {
		t.Fatalf("%d SetTempRateResponse records with the cycle off", n)
	}

	ps.SetBasalCycleEnabled(true)
	ps.RecordSetTempRateResponse(0, 5)
	got := records(ps, "SetTempRateResponse")
	if len(got) != 1 || got[0].Data["tempRateId"] != 5 {
		t.Errorf("SetTempRateResponse records = %+v, want one for temp rate 5", got)
	}
}

func TestTimeLeftIsInMillisecondsOfWholeMinutes(t *testing.T) {
	ps, clock := lifecyclePump(t)
	runTempRate(ps, 6, 30*time.Minute)
	clock.Advance(4*time.Minute + 30*time.Second)

	ps.EndTempRate(time.Time{})

	if got := records(ps, "TempRateCompleted")[0].Data["timeLeft"]; got != 25*60*1000 {
		t.Errorf("timeLeft = %v, want 25 whole minutes in ms", got)
	}
}
