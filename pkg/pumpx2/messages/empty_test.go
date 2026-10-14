package messages

import "testing"

// TestEmptyToString holds every class's field defaults, field initializers
// and no-arg constructor to what the jar's own no-arg instance prints.
func TestEmptyToString(t *testing.T) {
	for _, c := range Classes() {
		if c.EmptyToStringError != "" {
			continue
		}
		m, err := newInstance(c)
		if err != nil {
			t.Errorf("%s: no-arg constructor failed: %v", c.Name, err)
			continue
		}
		m.fillWithEmptyCargo()
		if got := m.ToString(); got != c.EmptyToString {
			t.Errorf("%s:\n  go:  %s\n  jar: %s", c.Name, got, c.EmptyToString)
		}
	}
}

// TestEveryClassIsPorted fails on a class the catalog has and no msg_*.go
// file registers, which a regenerated catalog from a newer jar would add.
func TestEveryClassIsPorted(t *testing.T) {
	for _, c := range Classes() {
		switch {
		case c.impl == nil:
			t.Errorf("%s has no Go port", c.Name)
		case c.impl.parse == nil && c.Name != "response.historyLog.HistoryLogStreamResponse":
			t.Errorf("%s has no Go port of parse()", c.Name)
		}
	}
}
