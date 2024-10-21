package bngsocket

// NewSafeChan erstellt einen neuen SafeChan mit dem angegebenen Puffer.
func NewSafeChan[T any]() *SafeChan[T] {
	return &SafeChan[T]{
		ch:     make(chan T),
		isOpen: true,
	}
}

func NewBufferdSafeChan[T any](buffSize int) *SafeChan[T] {
	return &SafeChan[T]{
		ch:     make(chan T, buffSize),
		isOpen: true,
	}
}

// Enter sendet einen Wert in den Kanal, wenn dieser offen ist.
func (sc *SafeChan[T]) Enter(value T) bool {
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
func (sc *SafeChan[T]) Close() {
	sc.mu.Lock()
	sc.isOpen = false
	close(sc.ch)
	sc.mu.Unlock()
}

// IsOpen gibt an ob der Chan geschlossen gewurden
func (sc *SafeChan[T]) IsOpen() bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.isOpen
}

// GetChannel gibt den Kanal zurück, um Werte zu empfangen.
func (sc *SafeChan[T]) Read() (T, bool) {
	return <-sc.ch, true
}
