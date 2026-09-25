package harness

import (
	"net/http"
	"strconv"

	"github.com/jwoglom/faketandem/pkg/state"
)

// defaultHistoryPageLimit is how many records GET /api/history returns when the
// caller names no limit, and maxHistoryPageLimit the ceiling on one it does.
const (
	defaultHistoryPageLimit = 200
	maxHistoryPageLimit     = 5000
)

// historyEntrySnapshot is one history-log record as the harness API reports it.
//
// The two timestamps are not two renderings of one number, and the difference
// is the whole point of having both:
//
//	Time      the record's TRUE instant on the pump's Clock, RFC3339 in UTC.
//	PumpSeconds  what actually went on the wire: pump-epoch seconds with the
//	          pump-clock skew and the pump's time zone applied.
//
// `/api/clock` reports both offsets, so either can be derived from the other:
//
//	unix(Time) = PumpSeconds + 1199145600 - pump_timezone_offset_seconds - pump_offset_seconds
type historyEntrySnapshot struct {
	Sequence uint32 `json:"sequence"`
	TypeID   int    `json:"type_id"`
	Type     string `json:"type"`
	Time     string `json:"time"`
	// PumpSeconds is the wire timestamp. `pump_time` is the same value under
	// the name the snapshot has always used it by, kept so an existing client
	// does not break.
	PumpSeconds uint32 `json:"pump_seconds"`
	PumpTime    uint32 `json:"pump_time"`
	// Data is the record's own wire fields, named as pumpX2 and TandemKit name
	// them. Nothing else is in here.
	Data map[string]interface{} `json:"data,omitempty"`
	// Extra is pump-side context the 26-byte record format has no room for
	// (a temp rate's delivered volume, the reason string behind a suspend, the
	// alert type behind an alarm). It never reaches the wire, and is kept
	// separate precisely so a consumer cannot mistake it for a record field.
	Extra map[string]interface{} `json:"extra,omitempty"`
	// SourceNibble is the high nibble of the record's type-ID word.
	SourceNibble uint8 `json:"source_nibble"`
}

// historySnapshot is the tail of the history log carried in GET /api/state.
type historySnapshot struct {
	Count         int                    `json:"count"`
	FirstSequence uint32                 `json:"first_sequence"`
	LastSequence  uint32                 `json:"last_sequence"`
	Entries       []historyEntrySnapshot `json:"entries"`
}

// historyPage is the body of GET /api/history.
type historyPage struct {
	// Count is how many records the log holds in total, FirstSequence and
	// LastSequence the range it spans -- both independent of this page.
	Count         int    `json:"count"`
	FirstSequence uint32 `json:"first_sequence"`
	LastSequence  uint32 `json:"last_sequence"`
	// Since and Limit echo the query that produced this page.
	Since uint32 `json:"since"`
	Limit int    `json:"limit"`
	// NextSince is what to pass as `since` to continue where this page stopped.
	NextSince uint32 `json:"next_since"`
	// Truncated reports that the limit cut the page short, i.e. there are more
	// records at or before LastSequence than were returned.
	Truncated bool                   `json:"truncated"`
	Entries   []historyEntrySnapshot `json:"entries"`
}

// handleHistory serves GET /api/history?since=<sequence>&limit=N.
//
// The 50-record tail in GET /api/state is a convenience, not a feed: a
// scenario that runs for a while loses records off the front of it without
// being told. This endpoint is the feed -- every record from `since` forward,
// with the sequence to resume from -- and it reports each record's fields, its
// non-wire context and its source nibble, which the tail did not.
func (h *Harness) handleHistory(w http.ResponseWriter, r *http.Request) {
	if h.pumpState == nil {
		writeError(w, http.StatusInternalServerError, "no pump state attached")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method %s not allowed on /api/history (use GET)", r.Method)
		return
	}

	query := r.URL.Query()
	since, err := parseUint32Param(query.Get("since"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid since: %v", err)
		return
	}
	limit := defaultHistoryPageLimit
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit %q (expected a positive integer)", raw)
			return
		}
		limit = parsed
	}
	if limit > maxHistoryPageLimit {
		limit = maxHistoryPageLimit
	}

	first, last := h.pumpState.GetHistoryLogSequenceRange()
	// `since` is exclusive: it is the last sequence the caller already has, so
	// a poller can hand back the previous page's next_since unchanged.
	from := since + 1
	if from < first {
		from = first
	}

	var entries []state.HistoryLogEntry
	if last >= from {
		entries = h.pumpState.GetHistoryLogEntries(from, last)
	}
	truncated := len(entries) > limit
	if truncated {
		entries = entries[:limit]
	}

	out := historyEntrySnapshots(entries)
	nextSince := since
	if n := len(out); n > 0 {
		nextSince = out[n-1].Sequence
	} else if last > nextSince {
		nextSince = last
	}

	writeJSON(w, http.StatusOK, historyPage{
		Count:         h.pumpState.GetHistoryLogCount(),
		FirstSequence: first,
		LastSequence:  last,
		Since:         since,
		Limit:         limit,
		NextSince:     nextSince,
		Truncated:     truncated,
		Entries:       out,
	})
}

// parseUint32Param reads an optional non-negative integer query parameter.
func parseUint32Param(raw string) (uint32, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(value), nil
}

// historyEntrySnapshots renders stored entries for the API.
func historyEntrySnapshots(entries []state.HistoryLogEntry) []historyEntrySnapshot {
	out := make([]historyEntrySnapshot, 0, len(entries))
	for _, e := range entries {
		out = append(out, historyEntrySnapshot{
			Sequence:     e.Sequence,
			TypeID:       e.TypeID,
			Type:         e.Type,
			Time:         formatTime(e.Timestamp),
			PumpSeconds:  e.PumpTime,
			PumpTime:     e.PumpTime,
			Data:         e.Data,
			Extra:        e.Extra,
			SourceNibble: e.SourceNibble,
		})
	}
	return out
}
