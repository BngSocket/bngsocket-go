package bngsocket

import (
	"sync"
)

// newSafeBool erstellt ein neues _SafeBool-Objekt mit dem angegebenen booleschen Wert.
// Dieses Objekt verwendet Synchronisationsmechanismen, um einen sicheren Zugriff auf den Wert zu gewährleisten.
func newSafeBool(v bool) _SafeBool {
	return _SafeBool{
		newSafeValue(v),
	}
}

// newSafeBytes erstellt ein neues _SafeBytes-Objekt mit den angegebenen Byte-Daten.
// Dieses Objekt verwendet Synchronisationsmechanismen, um einen sicheren Zugriff auf die Bytes zu gewährleisten.
func newSafeBytes(v []byte) _SafeBytes {
	return _SafeBytes{
		newSafeValue(v),
	}
}

// newSafeInt erstellt ein neues _SafeInt-Objekt mit dem angegebenen int-Wert.
// Dieses Objekt verwendet Synchronisationsmechanismen, um einen sicheren Zugriff auf den Wert zu gewährleisten.
func newSafeInt(v int) _SafeInt {
	return _SafeInt{
		newSafeValue(v),
	}
}

// newSafeValue erstellt ein neues _SafeValue-Objekt für einen generischen Typ T.
// Dieses Objekt verwendet einen Mutex und eine Bedingungsvariable, um einen sicheren Zugriff und Synchronisation zu gewährleisten.
func newSafeValue[T any](v T) _SafeValue[T] {
	mutex := new(sync.Mutex)
	return _SafeValue[T]{
		value:   &v,
		lock:    mutex,
		cond:    sync.NewCond(mutex),
		changes: 0,
	}
}

// newSafeAck erstellt ein neues _SafeAck-Objekt.
// Dieses Objekt verwendet einen sicheren Kanal für ACK-Rückmeldungen.
func newSafeAck() _SafeAck {
	return _SafeAck{
		_SafeChan: NewSafeChan[*_AckItem](),
	}
}

// newSafeMap erstellt ein neues _SafeMap-Objekt für die angegebenen Typen X und T.
// Dieses Objekt verwendet eine sync.Map, um einen sicheren Zugriff auf die enthaltenen Daten zu gewährleisten.
func newSafeMap[X any, T any]() _SafeMap[X, T] {
	return _SafeMap[X, T]{
		Map: new(sync.Map),
	}
}
