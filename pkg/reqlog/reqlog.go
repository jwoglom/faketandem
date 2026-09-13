// Package reqlog is the pump-side record of everything that crossed the link:
// every parsed inbound message, every outbound response and notification, and
// every connection event.
//
// It exists because an integration harness driving a driver against this
// emulator needs the pump's own account of the exchange to assert against --
// the equivalent of the driver's own sent-message spy, but from the other end,
// where it can also show what the pump decided not to send (a dropped
// response, a link cut mid-message). Without it a test can only observe the
// driver's view, which cannot distinguish "the pump never answered" from "the
// driver ignored the answer".
//
// Entries live in a fixed-size ring buffer with monotonically increasing
// sequence numbers, so a poller can ask for everything since the last sequence
// it saw and detect (via the reported first sequence still held) when it fell
// behind.
package reqlog

import (
	"sync"
	"time"
)

// DefaultCapacity is the number of entries retained when none is configured.
// A full pairing plus a sync is on the order of a hundred messages; 4096
// leaves room for a long scenario without unbounded growth.
const DefaultCapacity = 4096

// Entry kinds.
const (
	// KindRequest is a message received from the central and parsed.
	KindRequest = "request"
	// KindResponse is a message the pump sent in reply to a request.
	KindResponse = "response"
	// KindNotification is a message the pump sent unprompted (e.g. a
	// qualifying-event bitmask).
	KindNotification = "notification"
	// KindConnection is a link-level event: attach, detach, radio change.
	KindConnection = "connection"
	// KindFault is a record of a fault the fault injector applied.
	KindFault = "fault"
)

// Entry is one recorded event. Fields that do not apply to a kind are omitted.
type Entry struct {
	Seq  int       `json:"seq"`
	Time time.Time `json:"time"`
	Kind string    `json:"kind"`

	Characteristic string `json:"characteristic,omitempty"`
	Message        string `json:"message,omitempty"`
	Opcode         *int   `json:"opcode,omitempty"`
	TxID           *int   `json:"tx_id,omitempty"`

	// Cargo is the decoded field map, for inbound messages.
	Cargo map[string]interface{} `json:"cargo,omitempty"`
	// Fragments are the hex-encoded BLE fragments as they went on (or would
	// have gone on) the wire.
	Fragments []string `json:"fragments,omitempty"`
	// FragmentsSent is how many of Fragments actually reached the central. It
	// differs from len(Fragments) when a fault cut the link mid-message.
	FragmentsSent int `json:"fragments_sent,omitempty"`

	// Connected carries the new link state for KindConnection entries.
	Connected *bool `json:"connected,omitempty"`
	// Fault names the fault kind that produced or modified this entry.
	Fault string `json:"fault,omitempty"`
	// Note is free-form detail (a drop reason, an error, a delay applied).
	Note string `json:"note,omitempty"`
}

// Log is a bounded, concurrency-safe ring buffer of entries.
type Log struct {
	mu       sync.Mutex
	entries  []Entry
	capacity int
	// nextSeq is the sequence number the next appended entry gets. Sequence
	// numbers start at 1 and never restart, so 0 is always "before everything".
	nextSeq int
	// clock lets tests stamp entries deterministically.
	clock func() time.Time
}

// New creates a Log retaining at most capacity entries. A non-positive
// capacity uses DefaultCapacity.
func New(capacity int) *Log {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &Log{
		entries:  make([]Entry, 0, capacity),
		capacity: capacity,
		nextSeq:  1,
		clock:    time.Now,
	}
}

// SetClock overrides the timestamp source, so a harness running on a manual
// pump clock can have the log agree with the pump rather than with wall time.
func (l *Log) SetClock(now func() time.Time) {
	if now == nil {
		now = time.Now
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.clock = now
}

// Append records an entry, stamping its sequence number and time (unless the
// caller already set a time). It returns the assigned sequence number.
func (l *Log) Append(e Entry) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	e.Seq = l.nextSeq
	l.nextSeq++
	if e.Time.IsZero() {
		e.Time = l.clock()
	}

	if len(l.entries) == l.capacity {
		copy(l.entries, l.entries[1:])
		l.entries[len(l.entries)-1] = e
	} else {
		l.entries = append(l.entries, e)
	}
	return e.Seq
}

// Since returns every retained entry with a sequence number greater than seq,
// oldest first, along with the sequence number of the oldest entry still held
// (0 when the log is empty). A caller whose seq is below that number missed
// entries to the ring's turnover.
func (l *Log) Since(seq int) (entries []Entry, oldest int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.entries) > 0 {
		oldest = l.entries[0].Seq
	}
	for _, e := range l.entries {
		if e.Seq > seq {
			entries = append(entries, e)
		}
	}
	return entries, oldest
}

// All returns every retained entry, oldest first.
func (l *Log) All() []Entry {
	entries, _ := l.Since(0)
	return entries
}

// Len returns the number of retained entries.
func (l *Log) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.entries)
}

// LastSeq returns the sequence number of the most recently appended entry, or
// 0 if nothing has ever been appended.
func (l *Log) LastSeq() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.nextSeq - 1
}

// Clear discards every retained entry. Sequence numbers keep counting, so a
// client that clears and then polls with a stale sequence number sees only
// genuinely new entries.
func (l *Log) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = l.entries[:0]
}

// RecordRequest logs a parsed inbound message.
func (l *Log) RecordRequest(characteristic, message string, opcode, txID int, cargo map[string]interface{}, fragments []string) int {
	return l.Append(Entry{
		Kind:           KindRequest,
		Characteristic: characteristic,
		Message:        message,
		Opcode:         &opcode,
		TxID:           &txID,
		Cargo:          copyCargo(cargo),
		Fragments:      append([]string(nil), fragments...),
	})
}

// RecordResponse logs an outbound response: what was encoded, how many
// fragments actually went out, and any fault note explaining a difference.
func (l *Log) RecordResponse(characteristic, message string, opcode, txID int, fragments []string, fragmentsSent int, fault, note string) int {
	return l.Append(Entry{
		Kind:           KindResponse,
		Characteristic: characteristic,
		Message:        message,
		Opcode:         &opcode,
		TxID:           &txID,
		Fragments:      append([]string(nil), fragments...),
		FragmentsSent:  fragmentsSent,
		Fault:          fault,
		Note:           note,
	})
}

// RecordNotification logs an unprompted outbound payload, such as a
// qualifying-event bitmask.
func (l *Log) RecordNotification(characteristic, message, payloadHex string) int {
	return l.Append(Entry{
		Kind:           KindNotification,
		Characteristic: characteristic,
		Message:        message,
		Fragments:      []string{payloadHex},
		FragmentsSent:  1,
	})
}

// RecordConnection logs a link state change.
func (l *Log) RecordConnection(connected bool, note string) int {
	return l.Append(Entry{
		Kind:      KindConnection,
		Connected: &connected,
		Note:      note,
	})
}

// RecordFault logs that a fault was applied, for cases with no response entry
// of their own (a dropped request, a forced disconnect, a radio change).
func (l *Log) RecordFault(fault, message, note string) int {
	return l.Append(Entry{
		Kind:    KindFault,
		Fault:   fault,
		Message: message,
		Note:    note,
	})
}

func copyCargo(cargo map[string]interface{}) map[string]interface{} {
	if cargo == nil {
		return nil
	}
	out := make(map[string]interface{}, len(cargo))
	for k, v := range cargo {
		out[k] = v
	}
	return out
}
