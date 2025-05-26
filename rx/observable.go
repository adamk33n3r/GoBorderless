package rx

/**
 * I didn't like any of the existing Go Rx libraries, so I wrote my own simple one for my needs.
 */

import (
	"sync"
)

type Observable[T any] struct {
	iterable Iterable[T]
}

func (o *Observable[T]) Subscribe(cb func(item T)) {
	ch := o.iterable.Observe()
	go func() {
		for item := range ch {
			cb(item)
		}
	}()
}

type Iterable[T any] interface {
	Observe() <-chan T
	Subscribers() []chan T
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
