package bngsocket

import (
	"fmt"
	"reflect"
)

// NewSafeChan erstellt einen neuen _SafeChan mit dem angegebenen Puffer.
func NewSafeChan[T any]() *_SafeChan[T] {
	DebugPrint(fmt.Sprintf("New Safe Chan generated %s", reflect.TypeFor[T]().String()))
	return &_SafeChan[T]{
		ch:     make(chan T),
		isOpen: true,
	}
}

func NewBufferdSafeChan[T any](buffSize int) *_SafeChan[T] {
	DebugPrint(fmt.Sprintf("New Safe Bufferd Chan generated %s", reflect.TypeFor[T]().String()))
	return &_SafeChan[T]{
		ch:     make(chan T, buffSize),
		isOpen: true,
	}
}

// Enter sendet einen Wert in den Kanal, wenn dieser offen ist.
func (sc *_SafeChan[T]) Enter(value T) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Es wird geprüft ob der Chan geöffnet ist
	if !sc.isOpen {
		return false // Kanal ist geschlossen
	}

	// Es wird versucht den Wert in den Chan zu schreiben
	select {
	case sc.ch <- value:
	default:
		DebugPrint("Chan closed")
		return false
	}

	// Der Vorgang war erfolgreich
	return true
}

// Close schließt den Kanal.
func (sc *_SafeChan[T]) Destroy() {
	sc.mu.Lock()
	sc.isOpen = false
	close(sc.ch)
	sc.mu.Unlock()
	DebugPrint(fmt.Sprintf("Safe Chan destroyed %s", reflect.TypeFor[T]().String()))
}

// IsOpen gibt an ob der Chan geschlossen gewurden
func (sc *_SafeChan[T]) IsOpen() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.isOpen
}

// GetChannel gibt den Kanal zurück, um Werte zu empfangen.
func (sc *_SafeChan[T]) Read() (T, bool) {
	// Es wird ein Leehrer Wert erzeugt
	var nilSafeChanValue T

	// Es wird geprüft ob der _SafeChan geschlossen wurde
	if safeCahnIsClosed(sc) {
		return nilSafeChanValue, false
	}

	// Es wird entweder auf Daten gewartet oder darauf das der _SafeChan geschlossen wird
	r, ok := <-sc.ch
	if !ok {
		return nilSafeChanValue, false
	}

	// Die Daten werden zurückgegeben
	return r, true
}

// Gibt an ob das _SafeChan geschlossen wurde
func safeCahnIsClosed[T any](sc *_SafeChan[T]) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.isOpen || sc.ch == nil {
		return false
	}
	return true
}
