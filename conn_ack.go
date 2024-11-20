package bngsocket

import "sync"

// newConnACK erstellt ein neues _ConnACK-Objekt.
// Diese Funktion initialisiert die Synchronisationsmechanismen (Mutex und Bedingungsvariable)
// sowie den Anfangszustand des ACK-Handlers.
func newConnACK() *_ConnACK {
	n := new(_ConnACK)
	n.mutex = new(sync.Mutex)
	n.cond = sync.NewCond(n.mutex)
	n.state = 0
	return n
}

// WaitOfACK wartet auf ein ACK (Acknowledgment).
// Diese Methode blockiert, bis der Zustand des ACK-Handlers auf 1 gesetzt wird,
// was bedeutet, dass ein ACK empfangen wurde. Nach dem Empfangen wird der Zustand
// wieder auf 0 zurückgesetzt.
func (n *_ConnACK) WaitOfACK() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Warten, bis der Zustand ungleich null ist
	for n.state == 0 {
		n.cond.Wait()
	}

	// Der Zustand wird nach dem Empfang des ACK wieder auf 0 gesetzt
	n.state = 0

	return nil
}

// EnterACK signalisiert den Empfang eines ACK (Acknowledgment).
// Diese Methode setzt den Zustand des ACK-Handlers auf 1 und
// signalisiert allen wartenden Goroutinen, dass ein ACK eingegangen ist.
func (n *_ConnACK) EnterACK() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Zustand setzen, um den Empfang des ACK anzuzeigen
	n.state = 1

	// Alle wartenden Goroutinen signalisieren, dass ein ACK empfangen wurde
	n.cond.Broadcast()
	return nil
}
