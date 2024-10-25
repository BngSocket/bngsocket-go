package bngsocket

import (
	"errors"
	"fmt"
	"io"
	"syscall"
)

// Wird verwenet um beim Lessevorgang auf Fehler zu Reagieren
func readProcessErrorHandling(socket *BngConn, err error) {
	// Der Fehler wird ermittelt
	if errors.Is(err, io.EOF) {
		go fullCloseConn(socket)
		fmt.Println("readProcessErrorHandling_E")
		return
	} else if errors.Is(err, syscall.ECONNRESET) {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
		return
	} else if errors.Is(err, syscall.EPIPE) {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
		return
	} else {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
		return
	}
}

// Wird verwenet um beim Lessevorgang auf Fehler zu Reagieren
func writeProcessErrorHandling(socket *BngConn, err error) {
	// Der Fehler wird ermittelt
	if errors.Is(err, io.EOF) {
		// Die Verbindung wurde getrennt (EOF)
		fullCloseConn(socket)
		return
	} else if errors.Is(err, syscall.ECONNRESET) {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
		return
	} else if errors.Is(err, syscall.EPIPE) {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
		return
	} else {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
		return
	}
}

// Gibt an ob die Hintergrund Dauerschleifen eines Sockets aktiv sein sollen
func runningBackgroundServingLoop(ipcc *BngConn) bool {
	return !connectionIsClosed(ipcc)
}

// Gibt an ob eine Verbinding geschlossen wurde
func connectionIsClosed(ipcc *BngConn) bool {
	result := bool(
		ipcc.closed.Get() &&
			ipcc.closing.Get() &&
			ipcc.runningError.Get() != nil)
	return result
}

// Wird verwendet um den Socket vollständig zu schlißene
func fullCloseConn(s *BngConn) error {
	if connectionIsClosed(s) {
		return io.EOF
	}

	// DEBUG
	defer DebugPrint("Connection closed")

	// Es wird Markiert dass der Socket geschlossen ist
	s.closing.Set(true)

	// Der Socket wird geschlossen
	s.mu.Lock()
	closeerr := s.conn.Close()
	s.mu.Unlock()

	// Der Writeable Chan wird geschlossen
	s.writingChan.Destroy()

	// Es wird gewartet dass alle Hintergrundaufgaben abgeschlossen werden
	fmt.Println("WAIT OF END")
	s.backgroundProcesses.Wait()
	fmt.Println("ENDL_WAIT")

	// Es wird Signalisiert, dass die Verbindung final geschlossen wurde
	s.closed.Set(true)

	// Sollte ein Fehler vorhanden sein, wird dieser Zurückgegeben
	if closeerr != nil {
		return closeerr
	}

	// Es ist kein Fehler aufgetreten
	return nil
}
