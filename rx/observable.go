package rx

/**
 * I didn't like any of the existing Go Rx libraries, so I wrote my own simple one for my needs.
 */

import (
	"slices"
	"sync"
)

type Subscription struct {
	Unsubscribe func()
}

type Observable[T any] struct {
	iterable Iterable[T]
}

func (o *Observable[T]) Subscribe(cb func(item T)) Subscription {
	ch := o.iterable.Observe()
	go func() {
		for item := range ch {
			cb(item)
		}
	}()
	subscription := Subscription{
		Unsubscribe: func() {
			o.iterable.Unsub(ch)
		},
	}

	return subscription
}

func (o *Observable[T]) UnsubscribeAll() {
	for _, ch := range o.iterable.Subscribers() {
		o.iterable.Unsub(ch)
	}
}

type Iterable[T any] interface {
	Observe() <-chan T
	Subscribers() []chan T
	Unsub(<-chan T)
}
type IterableImpl[T any] struct {
	subscribers []chan T
	mutex       sync.RWMutex
}

func (i *IterableImpl[T]) Observe() <-chan T {
	ch := make(chan T)
	i.mutex.Lock()
	i.subscribers = append(i.subscribers, ch)
	i.mutex.Unlock()
	return ch
}

func (i *IterableImpl[T]) Unsub(ch <-chan T) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	idx := slices.IndexFunc(i.subscribers, func(c chan T) bool {
		return (<-chan T)(c) == ch
	})
	if idx == -1 {
		return
	}
	i.subscribers = append(i.subscribers[:idx], i.subscribers[idx+1:]...)
}

func (i *IterableImpl[T]) Subscribers() []chan T {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.subscribers
}

func newEventSourceIterable[T any](next <-chan T) Iterable[T] {
	it := &IterableImpl[T]{}

	go func() {
		for item := range next {
			for _, sub := range it.subscribers {
				sub <- item
			}
		}
	}()

	return it
}

func FromChannel[T any](ch <-chan T) Observable[T] {
	return Observable[T]{
		iterable: newEventSourceIterable(ch),
	}
}
