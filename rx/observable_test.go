package rx

import (
	"sync"
	"testing"
	"time"
)

// recv reads one item from the observable's callback stream, failing the test if
// nothing arrives in time. The observable is unbuffered end to end, so every
// assertion here needs a timeout rather than a bare receive.
func recv[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case item := <-ch:
		return item
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for observable to emit")
	}
	var zero T
	return zero
}

func expectNothing[T any](t *testing.T, ch <-chan T, why string) {
	t.Helper()
	select {
	case item := <-ch:
		t.Fatalf("%s: unexpectedly received %v", why, item)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestFromChannelDeliversToSubscriber(t *testing.T) {
	source := make(chan int)
	obs := FromChannel(source)

	got := make(chan int, 4)
	obs.Subscribe(func(item int) { got <- item })

	source <- 1
	source <- 2

	if v := recv(t, got); v != 1 {
		t.Errorf("first item = %d, want 1", v)
	}
	if v := recv(t, got); v != 2 {
		t.Errorf("second item = %d, want 2", v)
	}
}

// The window scanner pushes a slice of windows once a second and every open
// dialog is a subscriber, so fan-out to N subscribers is the core use case.
func TestFromChannelFansOutToAllSubscribers(t *testing.T) {
	source := make(chan string)
	obs := FromChannel(source)

	const subscribers = 3
	got := make([]chan string, subscribers)
	for i := range got {
		got[i] = make(chan string, 1)
		ch := got[i]
		obs.Subscribe(func(item string) { ch <- item })
	}

	source <- "hello"

	for i, ch := range got {
		if v := recv(t, ch); v != "hello" {
			t.Errorf("subscriber %d got %q, want %q", i, v, "hello")
		}
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	source := make(chan int)
	obs := FromChannel(source)

	kept := make(chan int, 1)
	obs.Subscribe(func(item int) { kept <- item })

	dropped := make(chan int, 1)
	sub := obs.Subscribe(func(item int) { dropped <- item })

	source <- 1
	recv(t, kept)
	recv(t, dropped)

	sub.Unsubscribe()

	source <- 2
	if v := recv(t, kept); v != 2 {
		t.Errorf("remaining subscriber got %d, want 2", v)
	}
	expectNothing(t, dropped, "after Unsubscribe")
}

func TestUnsubscribeAll(t *testing.T) {
	source := make(chan int)
	obs := FromChannel(source)

	// Three subscribers: iterating a live slice while Unsub mutates it would
	// skip or double-remove entries; Subscribers() must return a snapshot.
	got := make(chan int, 3)
	obs.Subscribe(func(item int) { got <- item })
	obs.Subscribe(func(item int) { got <- item })
	obs.Subscribe(func(item int) { got <- item })

	source <- 1
	recv(t, got)
	recv(t, got)
	recv(t, got)

	obs.UnsubscribeAll()

	// Nothing is listening any more, so the send must not block the producer.
	done := make(chan struct{})
	go func() {
		source <- 2
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("send blocked after UnsubscribeAll")
	}
	expectNothing(t, got, "after UnsubscribeAll")
}

func TestUnsubscribeTwiceIsSafe(t *testing.T) {
	source := make(chan int)
	obs := FromChannel(source)

	sub := obs.Subscribe(func(int) {})
	sub.Unsubscribe()
	sub.Unsubscribe() // must not panic or remove somebody else's channel
}

// The app-setting dialog subscribes while the scan loop is already emitting.
// Run with -race: this is what catches unsynchronised access to the subscriber
// list during fan-out.
func TestConcurrentSubscribeDuringEmission(t *testing.T) {
	source := make(chan int)
	obs := FromChannel(source)

	stop := make(chan struct{})
	var producer sync.WaitGroup
	producer.Add(1)
	go func() {
		defer producer.Done()
		for {
			select {
			case <-stop:
				return
			case source <- 1:
			}
		}
	}()

	var subs sync.WaitGroup
	for range 8 {
		subs.Add(1)
		go func() {
			defer subs.Done()
			received := make(chan struct{}, 1)
			sub := obs.Subscribe(func(int) {
				select {
				case received <- struct{}{}:
				default:
				}
			})
			<-received
			sub.Unsubscribe()
		}()
	}

	subs.Wait()
	close(stop)
	producer.Wait()
}

// Documents current behaviour: Unsub drops the channel from the fan-out list but
// never closes it, so the goroutine started by Subscribe stays parked forever.
// If subscription teardown is ever reworked, this test should start failing.
func TestUnsubscribeDoesNotCloseSubscriberChannel(t *testing.T) {
	source := make(chan int)
	obs := FromChannel(source)
	it := obs.iterable.(*IterableImpl[int])

	ch := it.Observe()
	if len(it.Subscribers()) != 1 {
		t.Fatalf("Subscribers() = %d, want 1", len(it.Subscribers()))
	}

	it.Unsub(ch)
	if len(it.Subscribers()) != 0 {
		t.Fatalf("Subscribers() after Unsub = %d, want 0", len(it.Subscribers()))
	}

	select {
	case _, open := <-ch:
		if !open {
			t.Fatal("channel was closed by Unsub; subscriber goroutine teardown changed")
		}
		t.Fatal("unsubscribed channel received an item")
	case <-time.After(100 * time.Millisecond):
		// Still open and idle, which is the documented (leaky) behaviour.
	}
}

func TestObserveRegistersDistinctChannels(t *testing.T) {
	it := &IterableImpl[int]{}
	a := it.Observe()
	b := it.Observe()

	if len(it.Subscribers()) != 2 {
		t.Fatalf("Subscribers() = %d, want 2", len(it.Subscribers()))
	}

	it.Unsub(a)
	remaining := it.Subscribers()
	if len(remaining) != 1 {
		t.Fatalf("Subscribers() after Unsub = %d, want 1", len(remaining))
	}
	if (<-chan int)(remaining[0]) != b {
		t.Error("Unsub removed the wrong channel")
	}
}

func TestUnsubUnknownChannelIsNoop(t *testing.T) {
	it := &IterableImpl[int]{}
	registered := it.Observe()

	stranger := make(chan int)
	it.Unsub((<-chan int)(stranger))

	remaining := it.Subscribers()
	if len(remaining) != 1 {
		t.Fatalf("Subscribers() = %d, want 1", len(remaining))
	}
	if (<-chan int)(remaining[0]) != registered {
		t.Error("Unsub of an unknown channel disturbed the subscriber list")
	}
}
