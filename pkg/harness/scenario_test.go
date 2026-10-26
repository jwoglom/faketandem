package harness

import (
	"fmt"
	"math"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/jwoglom/faketandem/pkg/state"
)

func scenarioHarness(t *testing.T) (*Harness, *state.PumpState, http.Handler) {
	t.Helper()
	h, ps, transport, mux := testHarness(t)
	transport.connected = false
	return h, ps, mux
}

func TestScenarioPresetReplaysHoursOfUseIntoHistoryIOBAndLastBolus(t *testing.T) {
	h, ps, _ := scenarioHarness(t)

	result, err := h.ApplyScenario(Scenario{Preset: "recent-doses"})
	if err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}

	if got := ps.Now(); !got.Equal(testInstant) {
		t.Errorf("the clock is at %s after the replay; want it back at %s", got, testInstant)
	}
	if len(result.Boluses) != 4 || len(result.Temps) != 2 {
		t.Fatalf("replayed %d boluses and %d temp rates; want 4 and 2", len(result.Boluses), len(result.Temps))
	}

	last, ok := ps.GetLastBolus()
	if !ok || last.BolusID != 4 || last.DeliveredUnits != 1.2 || last.EndReasonID != state.BolusEndReasonStopped {
		t.Errorf("last bolus = %+v (valid %v); want bolus 4 stopped at 1.2 U", last, ok)
	}
	if want := testInstant.Add(-30*time.Minute + 24*time.Second); last.EndTime.Sub(want).Abs() > time.Second {
		t.Errorf("last bolus ended at %s; want %s", last.EndTime, want)
	}

	temp := ps.GetTempRate()
	if !temp.Active || temp.Percent != 150 || !temp.EndTime.Equal(testInstant.Add(40*time.Minute)) {
		t.Errorf("temp rate = %+v; want 150%% running until 40 minutes from now", temp)
	}

	// Mudaliar, linear over the default profile's 5 hours: 3 U 200 minutes
	// ago, 1.5 U 75 and 1.2 U 30; the bolus 330 minutes ago has worn off.
	want := 3*(1-200.0/300) + 1.5*(1-75.0/300) + 1.2*(1-30.0/300)
	if iob := ps.GetIOB(); math.Abs(iob-want) > 0.02 {
		t.Errorf("IOB = %.3f U; want %.3f", iob, want)
	}
}

func TestScenarioWritesTheHistoryInTimeOrderWithABasalDeliveryEveryFiveMinutes(t *testing.T) {
	h, ps, _ := scenarioHarness(t)

	result, err := h.ApplyScenario(Scenario{Preset: "recent-doses"})
	if err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}

	first, last := ps.GetHistoryLogSequenceRange()
	entries := ps.GetHistoryLogEntries(first, last)
	var basal []state.HistoryLogEntry
	for i, e := range entries {
		if i > 0 && e.Timestamp.Before(entries[i-1].Timestamp) {
			t.Fatalf("record %d (%s) at %s is older than the one before it", e.Sequence, e.Type, e.Timestamp)
		}
		if e.TypeID == state.HistoryBasalDelivery {
			basal = append(basal, e)
		}
	}
	if start := testInstant.Add(-6 * time.Hour); len(basal) == 0 || !basal[0].Timestamp.Equal(start) {
		t.Fatalf("the first BasalDelivery is not at the window's start, %s: %v", start, basal)
	}
	// 73 cycles over 6 hours, and one more at the resume after a suspend that
	// spanned cycles.
	if len(basal) != 74 {
		t.Errorf("%d BasalDelivery records; want 74", len(basal))
	}
	if final := basal[len(basal)-1].Data; final["commandedRateSource"] != state.BasalSourceTemp {
		t.Errorf("the latest BasalDelivery runs %v; want the temp rate", final)
	}

	if len(result.BasalDeliveries) != len(basal) {
		t.Fatalf("the result lists %d BasalDelivery records; the log has %d", len(result.BasalDeliveries), len(basal))
	}
	rates := map[string]map[float64]bool{}
	for i, d := range result.BasalDeliveries {
		if d.Sequence != basal[i].Sequence {
			t.Fatalf("listed BasalDelivery %d is record %d; want %d", i, d.Sequence, basal[i].Sequence)
		}
		if rates[d.Source] == nil {
			rates[d.Source] = map[float64]bool{}
		}
		rates[d.Source][d.Rate] = true
	}
	want := map[string]map[float64]bool{"profile": {0.85: true}, "temp": {0: true, 1.275: true}, "suspended": {0: true}}
	if !reflect.DeepEqual(rates, want) {
		t.Errorf("listed BasalDelivery rates by source = %v; want %v", rates, want)
	}
}

func TestScenarioBasalProfileSetsTheRateOfEachBasalDelivery(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	minuteOfDay := func(at time.Time) int { return int(ps.PumpTimeFor(at)%86400) / 60 }
	start := minuteOfDay(testInstant.Add(-6 * time.Hour))
	boundary := (start + 3*60) / 30 * 30
	if boundary >= 24*60 || minuteOfDay(testInstant) < boundary {
		t.Fatalf("the window from minute %d crosses midnight; the test needs one that does not", start)
	}

	result, err := h.ApplyScenario(Scenario{Hours: 6, BasalProfile: []ScenarioBasalSegment{
		{Start: "00:00", Rate: 0.6},
		{Start: fmt.Sprintf("%02d:%02d", boundary/60, boundary%60), Rate: 1.2},
	}})
	if err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}

	counts := map[float64]int{}
	for _, d := range result.BasalDeliveries {
		at, _ := time.Parse(time.RFC3339Nano, d.Time)
		want := 0.6
		if minuteOfDay(at) >= boundary {
			want = 1.2
		}
		if d.Source != "profile" || d.Rate != want || d.ProfileRate != want {
			t.Errorf("BasalDelivery %d at minute %d = %+v; want the profile's %.1f U/hr", d.Sequence, minuteOfDay(at), d, want)
		}
		counts[d.Rate]++
	}
	if counts[0.6] == 0 || counts[1.2] == 0 {
		t.Errorf("BasalDelivery records by rate = %v; want both segments' rates", counts)
	}
	if got := result.Final.ProfileRate; got != 1.2 {
		t.Errorf("the profile rate after the replay is %.2f; want the later segment's 1.2", got)
	}
}

func TestScenarioEndpointRejectsABasalProfileItCannotStage(t *testing.T) {
	for name, body := range map[string]string{
		"not from midnight":  `{"basal_profile":[{"start":"01:00","rate":0.8}]}`,
		"out of order":       `{"basal_profile":[{"start":"00:00","rate":0.8},{"start":"08:00","rate":1},{"start":"06:00","rate":1}]}`,
		"not a time":         `{"basal_profile":[{"start":"midnight","rate":0.8}]}`,
		"with a basal_rate":  `{"basal_profile":[{"start":"00:00","rate":0.8}],"state":{"basal_rate":1}}`,
		"a rate over 15U/hr": `{"basal_profile":[{"start":"00:00","rate":16}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, _, mux := scenarioHarness(t)
			if status, resp := do(t, mux, http.MethodPost, "/api/state/scenario", body); status != http.StatusBadRequest {
				t.Errorf("status %d (%v); want 400", status, resp)
			}
		})
	}
}

func TestScenarioReplacesThePastItReplays(t *testing.T) {
	h, ps, _ := scenarioHarness(t)
	ps.SetIOB(5)
	ps.AddHistoryLogEntryAt(state.HistoryBolusCompleted, "BolusCompleted", testInstant, map[string]interface{}{"bolusId": 99})

	if _, err := h.ApplyScenario(Scenario{Preset: "scheduled-basal"}); err != nil {
		t.Fatalf("ApplyScenario: %v", err)
	}

	if iob := ps.GetIOB(); iob != 0 {
		t.Errorf("IOB = %.3f U after a scenario with no bolus; want 0", iob)
	}
	if _, ok := ps.GetLastBolus(); ok {
		t.Error("a last bolus survived a scenario with no bolus")
	}
	first, last := ps.GetHistoryLogSequenceRange()
	for _, e := range ps.GetHistoryLogEntries(first, last) {
		if e.TypeID == state.HistoryBolusCompleted {
			t.Fatalf("the record staged before the scenario is still in the log: %+v", e)
		}
	}
}

func TestScenarioEndpointRefusesWhileADriverIsConnected(t *testing.T) {
	_, _, _, mux := testHarness(t)

	status, body := do(t, mux, http.MethodPost, "/api/state/scenario", `{"preset":"recent-doses"}`)
	if status != http.StatusConflict {
		t.Fatalf("status %d (%v); want 409", status, body)
	}
}

func TestScenarioEndpointRejectsABolusThatStartsBeforeTheLastOneEnded(t *testing.T) {
	_, ps, mux := scenarioHarness(t)

	status, body := do(t, mux, http.MethodPost, "/api/state/scenario", `{"events":[
		{"minutes_ago":10,"type":"bolus","units":5,"duration_seconds":300},
		{"minutes_ago":8,"type":"bolus","units":1}]}`)
	if status != http.StatusBadRequest {
		t.Fatalf("status %d (%v); want 400", status, body)
	}
	if got := ps.Now(); !got.Equal(testInstant) {
		t.Errorf("a refused replay left the clock at %s; want %s", got, testInstant)
	}
}

func TestScenarioEndpointReportsWhatTheDriverWillRead(t *testing.T) {
	_, _, mux := scenarioHarness(t)

	status, body := do(t, mux, http.MethodPost, "/api/state/scenario", `{"name":"one bolus","hours":1,"events":[
		{"minutes_ago":30,"type":"bolus","units":2,"source":"remote"}]}`)
	if status != http.StatusOK {
		t.Fatalf("status %d (%v)", status, body)
	}
	scenario := body["scenario"].(map[string]interface{})
	final := scenario["final"].(map[string]interface{})
	if got := final["iob"].(float64); math.Abs(got-2*(1-30.0/300)) > 0.01 {
		t.Errorf("final IOB %.3f; want 1.8", got)
	}
	bolus := scenario["boluses"].([]interface{})[0].(map[string]interface{})
	if bolus["source_id"].(float64) != state.BolusSourceBluetoothRemote {
		t.Errorf("bolus %v; want source 8 (remote)", bolus)
	}
}
