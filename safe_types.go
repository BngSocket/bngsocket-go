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
