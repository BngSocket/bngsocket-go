package bngsocket

import (
	"errors"
	"fmt"
	"io"
	"syscall"
)

// Wird bei einem IO EOF Socket Fehler ausgeführt
func connIoEOFError(socket *BngConn) {
	// Es wird geprüft ob die Verbindung bereits geschlossen wurde
	if connectionIsClosed(socket) {
		return
	}

	// Die Verbindung wird als geschlossen Markeirt
	if socket.closed.Set(true) != 1 {
		return
	}

	// Der Fehler wird zwischengespeichert
	if socket.runningError.Set(io.EOF) != 1 {
		return
	}

	// DEBUG
	_DebugPrint("Connection closed")

	// Es werden alle Channel Listener Geschlossen
	for socket.openChannelListener.Count() != 0 {
		// Das Item wird Extrahiert
		value, found := socket.openChannelListener.PopFirst()
		if !found {
			break
		}

		// Der Channel wird geschlossen
		value.Close()
	}

	// Es werden alle Channel Listener Anfragen geschlossen
	for socket.openChannelJoinProcesses.Count() != 0 {
		// Das Item wird Extrahiert
		value, found := socket.openChannelJoinProcesses.PopFirst()
		if !found {
			break
		}

		// Der Vorgang wird verworfen
		close(value)
	}

	// Es werden alle Offenen Channel geschlossen
	for socket.openChannelInstances.Count() != 0 {
		// Das Item wird Extrahiert
		value, found := socket.openChannelInstances.PopFirst()
		if !found {
			break
		}

		// Der Vorgang wird verworfen
		value.Close()
	}

	// Es werden alle ausgehenden RPC Anfragen geschlossen
	for socket.openRpcRequests.Count() != 0 {

	}
}

// Wird verwenet um beim Lessevorgang auf Fehler zu Reagieren
func readProcessErrorHandling(socket *BngConn, err error) {
	// Der Fehler wird ermittelt
	if errors.Is(err, io.EOF) {
		connIoEOFError(socket)
	} else if errors.Is(err, syscall.ECONNRESET) {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
	} else if errors.Is(err, syscall.EPIPE) {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
	} else {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
	}
}

// Wird verwenet um beim Schreibvorgang auf Fehler zu Reagieren
func writeProcessErrorHandling(socket *BngConn, err error) {
	// Der Fehler wird ermittelt
	if errors.Is(err, io.EOF) {
		connIoEOFError(socket)
	} else if errors.Is(err, syscall.ECONNRESET) {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
	} else if errors.Is(err, syscall.EPIPE) {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
	} else {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
	}
}

// Gibt an ob die Hintergrund Dauerschleifen eines Sockets aktiv sein sollen
func runningBackgroundServingLoop(ipcc *BngConn) bool {
	return !connectionIsClosed(ipcc)
}

// Gibt an ob eine Verbinding geschlossen wurde
func connectionIsClosed(ipcc *BngConn) bool {
	if ipcc.closed.Get() {
		return true
	}
	if ipcc.closing.Get() {
		return true
	}
	if ipcc.runningError.Get() != nil {
		return true
	}
	return false
}

// Wird verwendet um den Socket vollständig zu schlißene
func fullCloseConn(s *BngConn) error {
	if connectionIsClosed(s) {
		return io.EOF
	}

	// DEBUG
	defer _DebugPrint("Connection closed")

	// Es wird Markiert dass der Socket geschlossen ist
	s.closing.Set(true)

	// Der Socket wird geschlossen
	s.connMutex.Lock()
	closeerr := s.conn.Close()
	s.connMutex.Unlock()

	// LOG
	_DebugPrint(fmt.Sprintf("BngConn(%s): CLOSE FULL", s._innerhid))

	// Es wird gewartet dass alle Hintergrundaufgaben abgeschlossen werden
	s.backgroundProcesses.Wait()

	// Es wird Signalisiert, dass die Verbindung final geschlossen wurde
	s.closed.Set(true)

	// Sollte ein Fehler vorhanden sein, wird dieser Zurückgegeben
	if closeerr != nil {
		fmt.Println("AA: " + closeerr.Error())
		return closeerr
	}

	// Es ist kein Fehler aufgetreten
	return nil
}
