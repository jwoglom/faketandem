package harness

import (
	"net/http"
	"testing"

	"github.com/jwoglom/faketandem/pkg/state"
)

func TestStatePutTurnsOnTheBasalCycleAndTheSnapshotReportsIt(t *testing.T) {
	_, ps, _, mux := testHarness(t)
	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)

	body := mustDo(t, mux, http.MethodPut, "/api/state",
		`{"basal_rate":1.0,"basal_cycle_enabled":true,"controliq_algorithm_rate":2.2,"cgm_available":false}`)

	applied := body["applied"].([]interface{})
	for _, key := range []string{"basal_cycle_enabled", "controliq_algorithm_rate", "cgm_available"} {
		if !containsString(applied, key) {
			t.Errorf("%q not applied: %v", key, applied)
		}
	}
	assertHistoryHas(t, ps, "BasalDelivery")

	cycle := fieldsOf(mustDo(t, mux, http.MethodGet, "/api/state", ""), "basal_cycle")
	if cycle["enabled"] != true || cycle["next_cycle"] != "2024-03-05T12:05:00Z" {
		t.Errorf("basal_cycle = %v, want enabled with the next 279 at 12:05", cycle)
	}
	if cycle["rate"] != 1.0 || cycle["source"] != 1.0 || cycle["controliq_algorithm_rate"] != 2.2 || cycle["cgm_available"] != false {
		t.Errorf("basal_cycle = %v", cycle)
	}

	mustDo(t, mux, http.MethodPost, "/api/clock/advance", `{"seconds":300}`)
	if cycle := fieldsOf(mustDo(t, mux, http.MethodGet, "/api/state", ""), "basal_cycle"); cycle["latest_cycle"] != "2024-03-05T12:05:00Z" {
		t.Errorf("latest_cycle = %v, want the 279 at 12:05", cycle["latest_cycle"])
	}
}

func TestTheFirstCycleSeesControlIQStagedInTheSamePut(t *testing.T) {
	_, ps, _, mux := testHarness(t)
	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)

	mustDo(t, mux, http.MethodPut, "/api/state",
		`{"basal_cycle_enabled":true,"closed_loop_enabled":true,"controliq_algorithm_rate":2.2}`)

	var first map[string]interface{}
	_, last := ps.GetHistoryLogSequenceRange()
	for _, entry := range ps.GetHistoryLogEntries(0, last) {
		if entry.Type == "BasalDelivery" {
			first = entry.Data
			break
		}
	}
	if first == nil {
		t.Fatal("no BasalDelivery record")
	}
	if first["commandedRateSource"] != state.BasalSourceControlIQ || first["commandedRate"] != 2200 {
		t.Errorf("first 279 = %v, want Control-IQ at 2200 mU/h", first)
	}
}

func TestAnAlarmSuspendCanKeepTheTemp(t *testing.T) {
	_, ps, _, mux := testHarness(t)
	mustDo(t, mux, http.MethodPut, "/api/clock", `{"mode":"manual","frozen":true,"now":"2024-03-05T12:00:00Z"}`)
	mustDo(t, mux, http.MethodPost, "/api/state/tempbasal/start", `{"percent":150,"duration_minutes":30}`)

	mustDo(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"alarm","keep_temp":true}`)

	if !ps.IsPumpingSuspended() || !ps.GetTempRate().Active {
		t.Fatal("want the pump suspended with its temp rate still in force")
	}
	if n := countEntries(ps, "TempRateCompleted"); n != 0 {
		t.Errorf("%d TempRateCompleted records, want none", n)
	}
	mustDo(t, mux, http.MethodPost, "/api/state/resume", `{}`)
	if !ps.GetTempRate().Active {
		t.Error("the temp rate did not run again after the resume")
	}
}

func TestKeepTempIsOnlyForAnAlarmSuspend(t *testing.T) {
	_, _, _, mux := testHarness(t)

	if code, _ := do(t, mux, http.MethodPost, "/api/state/suspend", `{"reason":"user","keep_temp":true}`); code != http.StatusBadRequest {
		t.Errorf("keep_temp on a user suspend = %d, want 400", code)
	}
}

func countEntries(ps *state.PumpState, entryType string) int {
	_, last := ps.GetHistoryLogSequenceRange()
	n := 0
	for _, entry := range ps.GetHistoryLogEntries(0, last) {
		if entry.Type == entryType {
			n++
		}
	}
	return n
}

func containsString(values []interface{}, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
