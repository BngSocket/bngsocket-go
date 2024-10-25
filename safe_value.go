package bngsocket

import (
	"fmt"
	"reflect"
	"sync"
)

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
	DebugPrint(fmt.Sprintf("New Safe Value generated %s", reflect.TypeFor[T]().String()))
	return SafeValue[T]{
		value:   &v,
		lock:    new(sync.Mutex),
		changes: 0,
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
