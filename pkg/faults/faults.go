// Package faults is the emulator's fault injector: a registry of armed
// failures that the response path and the transport consult before letting
// bytes reach the central.
//
// The faults here are the ones a driver's error handling is hardest to test
// against real hardware -- a response that never arrives although the pump
// applied the request, a link cut halfway through a multi-fragment message, a
// radio that goes away -- and each is scoped so a scenario can aim it at one
// message type and let everything else through.
//
// Nothing in this package knows about BLE, pumpX2 or HTTP: a fault is matched
// on an opcode, a message name and a characteristic name, which keeps it
// usable from both the router (message identity) and the transport
// (fragment-level) without an import cycle between them.
package faults

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Fault kinds.
const (
	// KindDropResponse suppresses the response. The handler still runs and its
	// state changes are still applied, which is the "applied but the answer
	// was lost" case a driver must recover from.
	KindDropResponse = "drop_response"
	// KindDelayResponse holds the response back for a configured duration.
	KindDelayResponse = "delay_response"
	// KindErrorResponse replaces the response with a protocol ErrorResponse.
	KindErrorResponse = "error_response"
	// KindDisconnect drops the link, either instead of answering the request
	// or partway through the response's fragments.
	KindDisconnect = "disconnect"
	// KindRejectQuickPair refuses a quick-pair reconnect, forcing the central
	// back into a full pairing.
	KindRejectQuickPair = "reject_quick_pair"
	// KindRadioOff makes the transport stop accepting connections and drops
	// any current one, modeling a pump whose radio went away.
	KindRadioOff = "radio_off"
	// KindRadioOn restores the radio.
	KindRadioOn = "radio_on"
)

// Disconnect timings.
const (
	// AfterRequest cuts the link instead of sending the response at all.
	AfterRequest = "request"
	// AfterPartialResponse cuts the link after FragmentsSent fragments of the
	// response have gone out.
	AfterPartialResponse = "partial_response"
)

// Fault is one armed failure.
type Fault struct {
	// ID identifies the fault for listing and clearing. Assigned on Arm.
	ID int `json:"id"`
	// Kind is one of the Kind* constants.
	Kind string `json:"kind"`

	// Opcode scopes the fault to one message opcode. Zero means any.
	Opcode int `json:"opcode,omitempty"`
	// Message scopes the fault to one message name (the request's name, as
	// cliparser reports it -- e.g. "SetTempRateRequest"). Empty means any.
	// Matching is case-insensitive and also accepts the response name, so
	// either end of a request/response pair can be named.
	Message string `json:"message,omitempty"`
	// Characteristic scopes the fault to one characteristic name. Empty means
	// any.
	Characteristic string `json:"characteristic,omitempty"`

	// Count is how many more times this fault fires. Zero with Every set means
	// "for every match"; zero without Every means the fault is spent.
	Count int `json:"count,omitempty"`
	// Every arms the fault for every match rather than the next Count.
	Every bool `json:"every,omitempty"`

	// DelayMs is the hold for KindDelayResponse.
	DelayMs int `json:"delay_ms,omitempty"`
	// ErrorCode is the code carried by KindErrorResponse.
	ErrorCode int `json:"error_code,omitempty"`
	// After selects when KindDisconnect cuts the link (AfterRequest or
	// AfterPartialResponse).
	After string `json:"after,omitempty"`
	// FragmentsSent is how many fragments go out before an
	// AfterPartialResponse disconnect.
	FragmentsSent int `json:"fragments_sent,omitempty"`

	// Fired counts how many times the fault has been applied.
	Fired int `json:"fired"`
}

// Delay returns the configured hold as a duration.
func (f *Fault) Delay() time.Duration {
	return time.Duration(f.DelayMs) * time.Millisecond
}

// Target describes the message a fault is being matched against.
type Target struct {
	Opcode         int
	Message        string
	Characteristic string
}

// Registry holds the armed faults. It is safe for concurrent use.
type Registry struct {
	mu     sync.Mutex
	faults []*Fault
	nextID int
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{nextID: 1}
}

// Arm validates and adds a fault, returning the stored copy with its ID.
func (r *Registry) Arm(f Fault) (*Fault, error) {
	if err := validate(&f); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	f.ID = r.nextID
	r.nextID++
	stored := &f
	r.faults = append(r.faults, stored)

	copied := *stored
	return &copied, nil
}

// validate normalizes a fault and rejects one that could never fire.
func validate(f *Fault) error {
	f.Kind = strings.TrimSpace(f.Kind)
	switch f.Kind {
	case KindDropResponse, KindErrorResponse, KindRejectQuickPair:
	case KindDelayResponse:
		if f.DelayMs <= 0 {
			return fmt.Errorf("%s requires a positive delay_ms", f.Kind)
		}
	case KindDisconnect:
		if f.After == "" {
			f.After = AfterRequest
		}
		if f.After != AfterRequest && f.After != AfterPartialResponse {
			return fmt.Errorf("disconnect after must be %q or %q, got %q", AfterRequest, AfterPartialResponse, f.After)
		}
		if f.After == AfterPartialResponse && f.FragmentsSent < 0 {
			return fmt.Errorf("fragments_sent must not be negative")
		}
	case KindRadioOff, KindRadioOn:
	case "":
		return fmt.Errorf("kind is required")
	default:
		return fmt.Errorf("unknown fault kind %q", f.Kind)
	}

	if f.Count < 0 {
		return fmt.Errorf("count must not be negative")
	}
	// "Next N" with no N given means the next one.
	if !f.Every && f.Count == 0 {
		f.Count = 1
	}
	return nil
}

// List returns copies of the armed faults, in arming order.
func (r *Registry) List() []Fault {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]Fault, 0, len(r.faults))
	for _, f := range r.faults {
		out = append(out, *f)
	}
	return out
}

// Clear removes every armed fault.
func (r *Registry) Clear() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := len(r.faults)
	r.faults = nil
	return n
}

// Remove deletes one fault by ID, reporting whether it was found.
func (r *Registry) Remove(id int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, f := range r.faults {
		if f.ID == id {
			r.faults = append(r.faults[:i], r.faults[i+1:]...)
			return true
		}
	}
	return false
}

// Match consumes and returns the first armed fault of any of the given kinds
// that applies to target, or nil. A "next N" fault's remaining count is
// decremented here and the fault is dropped once spent, so a caller must only
// call Match when it is actually going to apply the result.
func (r *Registry) Match(target Target, kinds ...string) *Fault {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, f := range r.faults {
		if !kindAllowed(f.Kind, kinds) || !matches(f, target) {
			continue
		}

		f.Fired++
		applied := *f

		if !f.Every {
			f.Count--
			if f.Count <= 0 {
				r.faults = append(r.faults[:i], r.faults[i+1:]...)
			}
		}
		return &applied
	}
	return nil
}

// Peek reports whether a fault of one of the given kinds applies to target,
// without consuming it.
func (r *Registry) Peek(target Target, kinds ...string) *Fault {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, f := range r.faults {
		if kindAllowed(f.Kind, kinds) && matches(f, target) {
			copied := *f
			return &copied
		}
	}
	return nil
}

func kindAllowed(kind string, kinds []string) bool {
	if len(kinds) == 0 {
		return true
	}
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

// matches reports whether a fault's scope covers target. An unset scope field
// matches anything.
func matches(f *Fault, t Target) bool {
	if f.Opcode != 0 && t.Opcode != 0 && f.Opcode != t.Opcode {
		return false
	}
	if f.Opcode != 0 && t.Opcode == 0 {
		return false
	}
	if f.Characteristic != "" && !strings.EqualFold(f.Characteristic, t.Characteristic) {
		return false
	}
	if f.Message == "" {
		return true
	}
	return messageMatches(f.Message, t.Message)
}

// messageMatches compares a scope's message name against a target's, treating
// the Request and Response halves of a pair as the same message: a scenario
// arming a fault for "SetTempRateRequest" means the exchange, and the response
// path only ever sees "SetTempRateResponse".
func messageMatches(scope, target string) bool {
	if strings.EqualFold(scope, target) {
		return true
	}
	return strings.EqualFold(baseName(scope), baseName(target))
}

func baseName(name string) string {
	for _, suffix := range []string{"Request", "Response"} {
		if len(name) > len(suffix) && strings.EqualFold(name[len(name)-len(suffix):], suffix) {
			return name[:len(name)-len(suffix)]
		}
	}
	return name
}
