package sse

import (
	"fmt"
	"testing"
	"time"
)

// drainOrdered reads n messages from the client and fails the test on the
// first out-of-order or missing message.
func drainOrdered(t *testing.T, cl Listener, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case msg := <-cl.Chan():
			want := fmt.Sprintf("data: %d\n\n", i)
			if msg.String() != want {
				t.Fatalf("message %d out of order: got %q, want %q", i, msg.String(), want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for message %d", i)
		}
	}
}

// TestBroadcastPreservesOrder guards the FIFO guarantee of the broadcast
// path: token-level SSE streams (agent chat deltas) are only meaningful in
// send order. A delivery worker pool with >1 goroutine reorders messages
// under scheduling pressure, which surfaced as interleaved/corrupted chat
// streams in the UI.
func TestBroadcastPreservesOrder(t *testing.T) {
	manager := NewManager(5)

	// Buffer must hold the full sequence: the non-blocking delivery drops
	// messages for slow clients, which is not what we test here.
	const total = 40 // fits within the client channel buffer (50)
	cl := NewClient("order-test")
	manager.Register(cl)

	for i := 0; i < total; i++ {
		manager.Send(NewMessage(fmt.Sprintf("%d", i)))
	}

	drainOrdered(t, cl, total)
}

// TestHistoryReplayedOnceWithoutDuplicates ensures a message entering the
// broadcast is recorded in history exactly once, independent of how many
// clients are connected (it used to be added once per delivered client).
func TestHistoryReplayedOnceWithoutDuplicates(t *testing.T) {
	manager := NewManager(1)

	a := NewClient("a")
	b := NewClient("b")
	manager.Register(a)
	manager.Register(b)

	manager.Send(NewMessage("only-once"))

	// Wait until both connected clients received the live message, which
	// proves the delivery goroutine has processed it into history too.
	for _, cl := range []Listener{a, b} {
		select {
		case <-cl.Chan():
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for live delivery")
		}
	}

	late := NewClient("late")
	manager.Register(late) // Register replays history

	got := 0
	for {
		select {
		case <-late.Chan():
			got++
		case <-time.After(200 * time.Millisecond):
			if got != 1 {
				t.Fatalf("late client received %d replayed messages, want 1", got)
			}
			return
		}
	}
}

// TestHistoryKeptWithoutClients ensures messages broadcast while nobody is
// connected still land in history and are replayed to the next client.
func TestHistoryKeptWithoutClients(t *testing.T) {
	manager := NewManager(1)

	manager.Send(NewMessage("while-alone"))

	// Give the delivery goroutine a moment to process the message.
	deadline := time.After(5 * time.Second)
	for {
		cl := NewClient("first")
		manager.Register(cl)
		select {
		case msg := <-cl.Chan():
			if msg.String() != "data: while-alone\n\n" {
				t.Fatalf("unexpected replayed message: %q", msg.String())
			}
			return
		case <-time.After(50 * time.Millisecond):
			manager.Unregister(cl.ID())
			select {
			case <-deadline:
				t.Fatal("message sent without clients never reached history")
			default:
			}
		}
	}
}
