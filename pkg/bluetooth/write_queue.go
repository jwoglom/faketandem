package bluetooth

import "sync"

type queuedWrite struct {
	characteristic CharacteristicType
	data           []byte
}

// writeQueue preserves BLE fragment order while allowing ATT writes to be
// acknowledged before the message handler does slow parsing or cryptography.
// Each connection owns a queue; stopping discards pending writes and waits for
// the current handler before authentication state is reset for the next peer.
type writeQueue struct {
	mu      sync.Mutex
	writes  chan queuedWrite
	stopped bool
	done    chan struct{}
}

func newWriteQueue(handler WriteHandler) *writeQueue {
	q := &writeQueue{writes: make(chan queuedWrite, 256), done: make(chan struct{})}
	go func() {
		defer close(q.done)
		for w := range q.writes {
			q.mu.Lock()
			stopped := q.stopped
			q.mu.Unlock()
			if stopped {
				return
			}
			handler(w.characteristic, w.data)
		}
	}()
	return q
}

// enqueue copies the gatt buffer, which is reused after the callback returns.
// A full queue is refused instead of blocking the ATT reader.
func (q *writeQueue) enqueue(charType CharacteristicType, data []byte) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.stopped {
		return false
	}
	select {
	case q.writes <- queuedWrite{charType, append([]byte(nil), data...)}:
		return true
	default:
		return false
	}
}

func (q *writeQueue) stop() {
	q.mu.Lock()
	if !q.stopped {
		q.stopped = true
		close(q.writes)
	}
	q.mu.Unlock()
	<-q.done
}
