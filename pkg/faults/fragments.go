package faults

import (
	"strings"
	"sync"
)

// FragmentDropper applies drop_fragment faults to the stream of BLE
// notification fragments leaving one transport.
//
// It lives here rather than in the transport because the transport must not
// know what a fault is, and here rather than in the router because the router
// works in whole messages: by the time a response reaches the router's send
// path it is already a list of fragments, but nothing above the transport sees
// them go out one at a time, which is what fragment-level loss needs.
//
// The only thing it reads out of a fragment is the first byte, the
// remaining-fragment counter that both pumpX2's PacketArrayList and TandemKit's
// BTResponseParser use to know when a message is complete. A fragment with a
// counter of zero ends its message, so the next one starts a new message at
// position 0 -- which is what makes "drop the second fragment of every message"
// expressible without the transport knowing anything about messages.
type FragmentDropper struct {
	registry *Registry

	mu sync.Mutex
	// next is the position the next fragment on each characteristic occupies
	// within its message.
	next map[string]int
	// seen counts fragments per characteristic, for every_nth.
	seen map[string]int
}

// NewFragmentDropper creates a dropper backed by registry. A nil registry
// drops nothing, so installing one unconditionally is safe.
func NewFragmentDropper(registry *Registry) *FragmentDropper {
	return &FragmentDropper{
		registry: registry,
		next:     make(map[string]int),
		seen:     make(map[string]int),
	}
}

// Allow reports whether a fragment should go out. It must be called exactly
// once per fragment, in send order, because it is also what advances the
// per-characteristic position and count.
//
// A dropped fragment still advances both: the central never sees it, but the
// pump did emit it, and a fault that shifted every subsequent fragment's
// position would make "drop fragment 1" mean something different on the next
// message.
func (d *FragmentDropper) Allow(characteristic string, fragment []byte) bool {
	if d == nil || d.registry == nil || len(fragment) == 0 {
		return true
	}

	d.mu.Lock()
	index := d.next[characteristic]
	d.seen[characteristic]++
	seq := d.seen[characteristic]
	if fragment[0] == 0 {
		// Last fragment of this message; the next one starts a new message.
		d.next[characteristic] = 0
	} else {
		d.next[characteristic] = index + 1
	}
	d.mu.Unlock()

	return d.registry.MatchFragment(characteristic, index, seq) == nil
}

// Reset forgets the per-characteristic positions and counts. Arming a
// drop_fragment fault calls it, so "every second fragment" means every second
// fragment from the moment the fault was armed rather than from whatever the
// running total happened to be, and so a fault armed mid-message starts
// counting positions at the next message rather than inside the current one.
func (d *FragmentDropper) Reset() {
	if d == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.next = make(map[string]int)
	d.seen = make(map[string]int)
}

// MatchFragment consumes and returns the drop_fragment fault that applies to a
// fragment at the given position, or nil when none does.
//
// index is the fragment's position within its message and seq is its 1-based
// position in the stream of fragments seen on that characteristic. As with
// Match, a "next N" fault's count is decremented here, so a caller must only
// call this when it is going to act on the answer.
func (r *Registry) MatchFragment(characteristic string, index, seq int) *Fault {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, f := range r.faults {
		if f.Kind != KindDropFragment {
			continue
		}
		if f.Characteristic != "" && !strings.EqualFold(f.Characteristic, characteristic) {
			continue
		}
		if !fragmentSelected(f, index, seq) {
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

// fragmentSelected reports whether a drop_fragment fault picks this fragment.
func fragmentSelected(f *Fault, index, seq int) bool {
	if f.Index != nil {
		return *f.Index == index
	}
	return f.EveryNth > 0 && seq%f.EveryNth == 0
}
