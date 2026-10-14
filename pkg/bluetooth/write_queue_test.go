package bluetooth

import (
	"testing"
	"time"
)

func TestWriteQueueAcknowledgesWhileHandlerBlocked(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	got := make(chan byte, 3)
	q := newWriteQueue(func(_ CharacteristicType, data []byte) {
		if data[0] == 1 {
			close(entered)
			<-release
		}
		got <- data[0]
	})
	defer q.stop()
	if !q.enqueue(CharAuthorization, []byte{1}) {
		t.Fatal("first write refused")
	}
	<-entered
	buffer := []byte{2}
	if !q.enqueue(CharControl, buffer) || !q.enqueue(CharAuthorization, []byte{3}) {
		t.Fatal("write refused while handler busy")
	}
	buffer[0] = 99 // gatt is free to reuse the original receive buffer
	close(release)
	for _, want := range []byte{1, 2, 3} {
		select {
		case value := <-got:
			if value != want {
				t.Fatalf("got fragment %d, want %d", value, want)
			}
		case <-time.After(time.Second):
			t.Fatal("queued write did not run")
		}
	}
}

func TestWriteQueueStopDiscardsPendingFragments(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	got := make(chan byte, 2)
	q := newWriteQueue(func(_ CharacteristicType, data []byte) {
		close(entered)
		<-release
		got <- data[0]
	})
	q.enqueue(CharAuthorization, []byte{1})
	<-entered
	q.enqueue(CharAuthorization, []byte{2})
	stopped := make(chan struct{})
	go func() { q.stop(); close(stopped) }()
	// Observe the close under the queue lock so this is independent of the
	// worker/stop goroutines' scheduling.
	deadline := time.Now().Add(time.Second)
	for {
		q.mu.Lock()
		closing := q.stopped
		q.mu.Unlock()
		if closing {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("stop did not close the queue")
		}
		time.Sleep(time.Millisecond)
	}
	if q.enqueue(CharAuthorization, []byte{3}) {
		t.Fatal("stopped queue accepted a write")
	}
	close(release)
	<-stopped
	if len(got) != 1 {
		t.Fatal("stop delivered fragments from the abandoned connection")
	}
}
