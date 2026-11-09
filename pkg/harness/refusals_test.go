package harness

import (
	"net/http"
	"testing"
)

func TestRefusalsCanBeSwitchedOff(t *testing.T) {
	_, ps, _, mux := testHarness(t)

	body := mustDo(t, mux, http.MethodPut, "/api/state", `{"refusals":{"bolus_while_busy":false}}`)

	if ps.GetRefusals().BolusWhileBusy || !ps.GetRefusals().TempRateWhileRunning {
		t.Errorf("refusals = %+v, want only bolus_while_busy off", ps.GetRefusals())
	}
	refusals := body["state"].(map[string]interface{})["refusals"].(map[string]interface{})
	if refusals["bolus_while_busy"] != false || refusals["temp_rate_under_control_iq"] != true {
		t.Errorf("snapshot refusals = %v", refusals)
	}
}
