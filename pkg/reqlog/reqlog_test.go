package reqlog

import (
	"testing"
	"time"
)

func TestAppendAssignsSequenceAndTime(t *testing.T) {
	l := New(10)
	stamp := time.Date(2024, time.March, 5, 12, 0, 0, 0, time.UTC)
	l.SetClock(func() time.Time { return stamp })

	first := l.Append(Entry{Kind: KindRequest, Message: "A"})
	second := l.Append(Entry{Kind: KindRequest, Message: "B"})

	if first != 1 || second != 2 {
		t.Fatalf("sequence numbers = %d, %d; want 1, 2", first, second)
	}
	if got := l.LastSeq(); got != 2 {
		t.Errorf("LastSeq() = %d, want 2", got)
	}

	entries := l.All()
	if len(entries) != 2 {
		t.Fatalf("retained %d entries, want 2", len(entries))
	}
	if !entries[0].Time.Equal(stamp) {
		t.Errorf("entry time = %v, want the injected clock's %v", entries[0].Time, stamp)
	}
	if entries[0].Message != "A" || entries[1].Message != "B" {
		t.Errorf("entries out of order: %q then %q", entries[0].Message, entries[1].Message)
	}
}

func TestSinceReturnsOnlyNewerEntries(t *testing.T) {
	l := New(10)
	for _, name := range []string{"A", "B", "C"} {
		l.Append(Entry{Kind: KindRequest, Message: name})
	}

	entries, oldest := l.Since(2)
	if oldest != 1 {
		t.Errorf("oldest = %d, want 1", oldest)
	}
	if len(entries) != 1 || entries[0].Message != "C" {
		t.Fatalf("Since(2) returned %+v, want just C", entries)
	}

	if entries, _ := l.Since(3); len(entries) != 0 {
		t.Errorf("Since(last) returned %d entries, want none", len(entries))
	}
}

func TestRingDropsOldestAndReportsIt(t *testing.T) {
	l := New(3)
	for _, name := range []string{"A", "B", "C", "D", "E"} {
		l.Append(Entry{Kind: KindRequest, Message: name})
	}

	if got := l.Len(); got != 3 {
		t.Fatalf("retained %d entries, want the 3-entry capacity", got)
	}

	entries, oldest := l.Since(0)
	if oldest != 3 {
		t.Errorf("oldest = %d, want 3 (A and B turned over)", oldest)
	}
	if len(entries) != 3 || entries[0].Message != "C" || entries[2].Message != "E" {
		t.Fatalf("retained %+v, want C, D, E", entries)
	}
	// A poller that asked for everything after entry 1 can tell it fell
	// behind, because the oldest retained sequence is above its cursor.
	if oldest <= 1 {
		t.Error("a caller polling since=1 could not detect the turnover")
	}
}

func TestClearKeepsSequenceNumbersMovingForward(t *testing.T) {
	l := New(10)
	l.Append(Entry{Kind: KindRequest, Message: "A"})
	l.Clear()

	if got := l.Len(); got != 0 {
		t.Fatalf("Len() = %d after Clear", got)
	}
	seq := l.Append(Entry{Kind: KindRequest, Message: "B"})
	if seq != 2 {
		t.Errorf("sequence after Clear = %d, want 2 so a stale cursor cannot re-read old entries", seq)
	}
}

func TestRecordHelpersCaptureTheFields(t *testing.T) {
	l := New(10)

	l.RecordRequest("CurrentStatus", "ApiVersionRequest", 32, 7, map[string]interface{}{"a": 1}, []string{"aa"})
	l.RecordResponse("CurrentStatus", "ApiVersionResponse", 33, 7, []string{"bb", "cc"}, 1, "disconnect", "cut short")
	l.RecordNotification("QualifyingEvents", "QualifyingEvents(0x400)", "00040000")
	l.RecordConnection(true, "attached")
	l.RecordFault("drop_response", "SetTempRateRequest", "armed")

	entries := l.All()
	if len(entries) != 5 {
		t.Fatalf("recorded %d entries, want 5", len(entries))
	}

	req := entries[0]
	if req.Kind != KindRequest || req.Opcode == nil || *req.Opcode != 32 || req.TxID == nil || *req.TxID != 7 {
		t.Errorf("request entry = %+v", req)
	}
	if req.Cargo["a"] != 1 {
		t.Errorf("request cargo = %v", req.Cargo)
	}

	resp := entries[1]
	if resp.FragmentsSent != 1 || len(resp.Fragments) != 2 {
		t.Errorf("response entry should show 1 of 2 fragments sent, got %d of %d",
			resp.FragmentsSent, len(resp.Fragments))
	}
	if resp.Fault != "disconnect" {
		t.Errorf("response fault = %q", resp.Fault)
	}

	if entries[2].Kind != KindNotification {
		t.Errorf("notification kind = %q", entries[2].Kind)
	}
	if entries[3].Connected == nil || !*entries[3].Connected {
		t.Errorf("connection entry = %+v", entries[3])
	}
	if entries[4].Kind != KindFault {
		t.Errorf("fault kind = %q", entries[4].Kind)
	}
}

func TestRecordRequestCopiesCargo(t *testing.T) {
	l := New(10)
	cargo := map[string]interface{}{"units": 1}
	l.RecordRequest("Control", "InitiateBolusRequest", 64, 1, cargo, nil)

	// Mutating the caller's map must not rewrite history.
	cargo["units"] = 99

	if got := l.All()[0].Cargo["units"]; got != 1 {
		t.Errorf("logged cargo changed under us: units = %v, want 1", got)
	}
}
