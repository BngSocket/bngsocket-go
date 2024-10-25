package bngsocket

import (
	"sync"
)

type SafeChan[T any] struct {
	ch     chan T
	mu     sync.Mutex
	isOpen bool
}

type SafeValue[T any] struct {
	value   *T
	changes uint64
	lock    *sync.Mutex
}

type SafeInt struct {
	SafeValue[int]
}

type SafeBool struct {
	SafeValue[bool]
}

type SafeBytes struct {
	SafeValue[[]byte]
}

type AckItem struct {
	pid   uint64
	state uint8
}

type SafeAck struct {
	*SafeChan[*AckItem]
}

type SafeMap[X any, T any] struct {
	*sync.Map
}

func (t *SafeValue[T]) Set(v T) uint64 {
	t.lock.Lock()
	t.value = &v
	t.changes = t.changes + 1
	newtValue := t.changes
	t.lock.Unlock()
	return newtValue
}

func (t *SafeValue[T]) Get() (v T) {
	t.lock.Lock()
	v = *t.value
	t.lock.Unlock()
	return
}

func (t *SafeInt) Add(val int) {
	t.lock.Lock()
	cint := *t.value
	added := cint + val
	t.value = &added
	t.lock.Unlock()
}

func (t *SafeInt) Sub(val int) {
	t.lock.Lock()
	cint := *t.value
	subtracted := cint - val
	t.value = &subtracted
	t.lock.Unlock()
}

func (t *SafeMap[X, T]) Store(key X, val T) {
	t.Map.Store(key, val)
}

func (t *SafeMap[X, T]) Delete(key X) {
	t.Map.Delete(key)
}

func (t *SafeMap[X, T]) Load(key X) (T, bool) {
	r, ok := t.Map.Load(key)
	if !ok {
		var zeroValue T
		return zeroValue, false
	}
	conv := r.(T)
	return conv, true
}
