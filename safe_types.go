package bngsocket

import (
	"sync"
)

type _SafeChan[T any] struct {
	ch     chan T
	mu     sync.Mutex
	isOpen bool
}

type _SafeValue[T any] struct {
	value   *T
	changes uint64
	lock    *sync.Mutex
	cond    *sync.Cond
}

type _SafeInt struct {
	_SafeValue[int]
}

type _SafeBool struct {
	_SafeValue[bool]
}

type _SafeBytes struct {
	_SafeValue[[]byte]
}

type _AckItem struct {
	pid   uint64
	state uint8
}

type _SafeAck struct {
	*_SafeChan[*_AckItem]
}

type _SafeMap[X any, T any] struct {
	*sync.Map
}

func (t *_SafeValue[T]) Set(v T) uint64 {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.value = &v
	t.changes = t.changes + 1
	newtValue := t.changes
	t.cond.Broadcast()
	return newtValue
}

func (t *_SafeValue[T]) Get() (v T) {
	t.lock.Lock()
	v = *t.value
	t.lock.Unlock()
	return
}

func (t *_SafeValue[T]) Watch() {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.cond.Wait() // Wartet auf Benachrichtigung, dass sich der Wert geändert hat
}

func (t *_SafeInt) Add(val int) {
	t.lock.Lock()
	cint := *t.value
	added := cint + val
	t.value = &added
	t.lock.Unlock()
}

func (t *_SafeInt) Sub(val int) {
	t.lock.Lock()
	cint := *t.value
	subtracted := cint - val
	t.value = &subtracted
	t.lock.Unlock()
}

func (t *_SafeMap[X, T]) Store(key X, val T) {
	t.Map.Store(key, val)
}

func (t *_SafeMap[X, T]) Delete(key X) {
	t.Map.Delete(key)
}

func (t *_SafeMap[X, T]) Load(key X) (T, bool) {
	r, ok := t.Map.Load(key)
	if !ok {
		var zeroValue T
		return zeroValue, false
	}
	conv := r.(T)
	return conv, true
}

func (t *_SafeMap[X, T]) Count() int {
	count := 0
	t.Map.Range(func(key, value any) bool {
		count++
		return true // Weitermachen, um alle Elemente zu zählen
	})
	return count
}

func (t *_SafeMap[X, T]) PopFirst() (value T, found bool) {
	found = false
	// Verwende die Range-Methode, um das erste Element zu finden
	t.Map.Range(func(k, v any) bool {
		// Setze das erste gefundene Element und markiere als gefunden
		value, found = v.(T), true
		// Entferne das Element aus der Map
		t.Map.Delete(k)
		// Rückgabe von false, um die Range-Schleife zu beenden
		return false
	})
	return value, found
}
