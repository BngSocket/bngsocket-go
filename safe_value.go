package bngsocket

import "sync"

func newSafeBool(v bool) SafeBool {
	return SafeBool{
		newSafeValue(v),
	}
}

func newSafeBytes(v []byte) SafeBytes {
	return SafeBytes{
		newSafeValue(v),
	}
}

func newSafeInt(v int) SafeInt {
	return SafeInt{
		newSafeValue(v),
	}
}

func newSafeValue[T any](v T) SafeValue[T] {
	return SafeValue[T]{
		value: &v,
		lock:  new(sync.Mutex),
	}
}

func newSafeAck() SafeAck {
	return SafeAck{
		SafeChan: NewSafeChan[*AckItem](),
	}
}

func newSafeMap[X any, T any]() SafeMap[X, T] {
	return SafeMap[X, T]{
		Map: new(sync.Map),
	}
}

func (t *SafeValue[T]) Set(v T) {
	t.lock.Lock()
	t.value = &v
	t.lock.Unlock()
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
