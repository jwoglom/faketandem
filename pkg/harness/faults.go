package harness

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/jwoglom/faketandem/pkg/bluetooth"
	"github.com/jwoglom/faketandem/pkg/faults"

	log "github.com/sirupsen/logrus"
)

// handleFaults serves /api/faults and /api/faults/{id}.
func (h *Harness) handleFaults(w http.ResponseWriter, r *http.Request) {
	if h.registry == nil {
		writeError(w, http.StatusInternalServerError, "no fault registry attached")
		return
	}

	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/faults"), "/")

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]interface{}{"faults": h.registry.List()})

	case http.MethodPost:
		h.armFault(w, r)

	case http.MethodDelete:
		h.clearFaults(w, id)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method %s not allowed on /api/faults", r.Method)
	}
}

// armFault adds a fault. Radio faults are not stored: they are a state change
// of the transport, applied immediately, because "the radio is off" is not
// something that happens to the next message -- it is a condition.
func (h *Harness) armFault(w http.ResponseWriter, r *http.Request) {
	var f faults.Fault
	if err := decodeBody(r, &f); err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	switch f.Kind {
	case faults.KindRadioOff, faults.KindRadioOn:
		h.applyRadioFault(w, f.Kind)
		return
	}

	armed, err := h.registry.Arm(f)
	if err != nil {
		writeError(w, http.StatusBadRequest, "%v", err)
		return
	}

	log.Infof("harness: armed fault %d: kind=%s message=%q opcode=%d every=%v count=%d",
		armed.ID, armed.Kind, armed.Message, armed.Opcode, armed.Every, armed.Count)
	if h.requests != nil {
		h.requests.RecordFault(armed.Kind, armed.Message, "armed")
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"fault": armed, "faults": h.registry.List()})
}

// applyRadioFault turns the transport's radio off or on.
func (h *Harness) applyRadioFault(w http.ResponseWriter, kind string) {
	radio, ok := h.transport.(bluetooth.RadioController)
	if !ok {
		writeError(w, http.StatusNotImplemented,
			"this transport cannot switch its radio (only the virtual transport can)")
		return
	}

	enabled := kind == faults.KindRadioOn
	radio.SetRadioEnabled(enabled)
	if h.requests != nil {
		h.requests.RecordFault(kind, "", "radio "+map[bool]string{true: "on", false: "off"}[enabled])
	}
	log.Infof("harness: radio turned %s", map[bool]string{true: "on", false: "off"}[enabled])

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"radio_enabled": radio.RadioEnabled(),
		"connected":     h.transport.IsConnected(),
	})
}

// clearFaults removes one fault by id, or every armed fault.
func (h *Harness) clearFaults(w http.ResponseWriter, id string) {
	if id == "" {
		cleared := h.registry.Clear()
		log.Infof("harness: cleared %d armed fault(s)", cleared)
		writeJSON(w, http.StatusOK, map[string]interface{}{"cleared": cleared, "faults": h.registry.List()})
		return
	}

	parsed, err := strconv.Atoi(id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid fault id %q: %v", id, err)
		return
	}
	if !h.registry.Remove(parsed) {
		writeError(w, http.StatusNotFound, "no armed fault with id %d", parsed)
		return
	}

	log.Infof("harness: cleared fault %d", parsed)
	writeJSON(w, http.StatusOK, map[string]interface{}{"cleared": 1, "faults": h.registry.List()})
}
