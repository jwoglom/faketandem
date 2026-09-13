package harness

import (
	"net/http"
	"strconv"

	"github.com/jwoglom/faketandem/pkg/reqlog"

	log "github.com/sirupsen/logrus"
)

// logResponse is the body of GET /api/log.
type logResponse struct {
	// Entries are the records newer than the requested sequence, oldest first.
	Entries []reqlog.Entry `json:"entries"`
	// Oldest is the sequence number of the oldest record still retained. A
	// caller polling with a "since" below it lost records to the ring's
	// turnover and should treat its view as incomplete.
	Oldest int `json:"oldest_seq"`
	// LastSeq is the sequence number of the newest record, to poll from next.
	LastSeq int `json:"last_seq"`
	// Retained is how many records the log currently holds.
	Retained int `json:"retained"`
}

func (h *Harness) handleLog(w http.ResponseWriter, r *http.Request) {
	if h.requests == nil {
		writeError(w, http.StatusInternalServerError, "no request log attached")
		return
	}

	switch r.Method {
	case http.MethodGet:
		since := 0
		if raw := r.URL.Query().Get("since"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid since=%q: %v", raw, err)
				return
			}
			since = parsed
		}

		entries, oldest := h.requests.Since(since)
		if entries == nil {
			entries = []reqlog.Entry{}
		}
		writeJSON(w, http.StatusOK, logResponse{
			Entries:  entries,
			Oldest:   oldest,
			LastSeq:  h.requests.LastSeq(),
			Retained: h.requests.Len(),
		})

	case http.MethodDelete:
		h.requests.Clear()
		log.Info("harness: request log cleared")
		writeJSON(w, http.StatusOK, logResponse{
			Entries: []reqlog.Entry{},
			LastSeq: h.requests.LastSeq(),
		})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method %s not allowed on /api/log", r.Method)
	}
}
