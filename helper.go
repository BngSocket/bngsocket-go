package bngsocket

import (
	"sync"
)

// Gibt an ob die Bng Verbindung geschlossen wurde
func IsConnectionClosed(conn *BngConn) bool {
	return connectionIsClosed(conn)
}

// Wartet darauf dass sich der Status einer Bng Verbindung ändert
func MonitorConnection(conn *BngConn) error {
	done := make(chan error, 1) // Kanal für den Rückgabefehler
	var once sync.Once          // Sichert ab, dass der Kanal nur einmal geschlossen wird

	// Starte eine Goroutine für runningError, die spezifischen Fehler zurückgibt
	go func() {
		conn.runningError.Watch()
		once.Do(func() { done <- conn.runningError.Get() }) // Gibt den spezifischen Fehler zurück
	}()

	// Starte eine Goroutine für closing, die io.EOF zurückgibt
	go func() {
		conn.closing.Watch()
		once.Do(func() { done <- ErrConnectionClosedEOF }) // Gibt einen allgemeinen IO-Fehler zurück
	}()

	// Starte eine Goroutine für closed, die io.EOF zurückgibt
	go func() {
		conn.closed.Watch()
		once.Do(func() { done <- ErrConnectionClosedEOF }) // Gibt einen allgemeinen IO-Fehler zurück
	}()

	// Blockiert, bis eine Änderung erkannt wird, und gibt dann den Fehler zurück
	return <-done
}
