package bngsocket

import (
	"errors"
	"fmt"
	"io"
	"syscall"
)

// connIoEOFError wird bei einem IO EOF Socket Fehler ausgeführt.
// Diese Funktion behandelt das Auftreten eines EOF-Fehlers auf der Socket-Verbindung.
// Sie überprüft zunächst, ob die Verbindung bereits geschlossen wurde. Falls nicht, markiert
// sie die Verbindung als geschlossen und speichert den EOF-Fehler. Anschließend werden alle
// geöffneten Channel Listener, Channel Join Prozesse, offenen Channel Instanzen und ausgehenden
// RPC-Anfragen geschlossen oder verworfen. Abschließend wird eine Debug-Nachricht ausgegeben.
//
// Parameter:
//   - socket *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und
//     zugehörige Ressourcen verwaltet.
func connIoEOFError(socket *BngConn) {
	// Es wird geprüft, ob die Verbindung bereits geschlossen wurde
	if connectionIsClosed(socket) {
		return
	}

	// Die Verbindung wird als geschlossen markiert
	if socket.closed.Set(true) != 1 {
		return
	}

	// Der Fehler wird zwischengespeichert
	if socket.runningError.Set(io.EOF) != 1 {
		return
	}

	// DEBUG: Verbindung wurde geschlossen
	_DebugPrint("Connection closed")

	// Alle Channel Listener werden geschlossen
	for socket.openChannelListener.Count() != 0 {
		// Das erste Item wird extrahiert
		value, found := socket.openChannelListener.PopFirst()
		if !found {
			break
		}

		// Der Channel wird geschlossen
		value.Close()
	}

	// Alle Channel Listener Anfragen werden geschlossen
	for socket.openChannelJoinProcesses.Count() != 0 {
		// Das erste Item wird extrahiert
		value, found := socket.openChannelJoinProcesses.PopFirst()
		if !found {
			break
		}

		// Der Vorgang wird verworfen
		close(value)
	}

	// Alle offenen Channel werden geschlossen
	for socket.openChannelInstances.Count() != 0 {
		// Das erste Item wird extrahiert
		value, found := socket.openChannelInstances.PopFirst()
		if !found {
			break
		}

		// Der Channel wird geschlossen
		value.Close()
	}

	// Alle ausgehenden RPC Anfragen werden geschlossen
	for socket.openRpcRequests.Count() != 0 {
		// Implementierung fehlt: Hier sollten die ausgehenden RPC Anfragen geschlossen werden
		// Beispiel:
		// rpcRequest, found := socket.openRpcRequests.PopFirst()
		// if found {
		//     rpcRequest.Close()
		// }
	}
}

// readProcessErrorHandling wird verwendet, um beim Lesvorgang auf Fehler zu reagieren.
// Diese Funktion analysiert den aufgetretenen Fehler und führt entsprechende Maßnahmen durch.
// Bei einem EOF-Fehler wird connIoEOFError aufgerufen. Bei ECONNRESET oder EPIPE Fehlern wird
// die Methode _ConsensusProtocolTermination mit einer entsprechenden Fehlermeldung aufgerufen.
// Für alle anderen Fehler wird ebenfalls _ConsensusProtocolTermination aufgerufen.
// Die Funktion gibt einen booleschen Wert zurück, der angibt, ob die Verbindung geschlossen wurde.
//
// Parameter:
//   - socket *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und
//     zugehörige Ressourcen verwaltet.
//   - err error: Der aufgetretene Fehler, der verarbeitet werden soll.
//
// Rückgabe:
//   - bool: true, wenn die Verbindung geschlossen wurde, andernfalls false.
func readProcessErrorHandling(socket *BngConn, err error) bool {
	// Der Fehler wird ermittelt und entsprechende Maßnahmen werden ergriffen
	if errors.Is(err, io.EOF) {
		connIoEOFError(socket)
	} else if errors.Is(err, syscall.ECONNRESET) {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
	} else if errors.Is(err, syscall.EPIPE) {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
	} else {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
	}
	return true
}

// writeProcessErrorHandling wird verwendet, um beim Schreibvorgang auf Fehler zu reagieren.
// Diese Funktion analysiert den aufgetretenen Fehler und führt entsprechende Maßnahmen durch.
// Bei einem EOF-Fehler wird connIoEOFError aufgerufen. Bei ECONNRESET oder EPIPE Fehlern wird
// die Methode _ConsensusProtocolTermination mit einer entsprechenden Fehlermeldung aufgerufen.
// Für alle anderen Fehler wird ebenfalls _ConsensusProtocolTermination aufgerufen.
//
// Parameter:
//   - socket *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und
//     zugehörige Ressourcen verwaltet.
//   - err error: Der aufgetretene Fehler, der verarbeitet werden soll.
//
// Rückgabe:
//   - bool: true, wenn die Verbindung geschlossen wurde, andernfalls false.
func writeProcessErrorHandling(socket *BngConn, err error) bool {
	// Der Fehler wird ermittelt und entsprechende Maßnahmen werden ergriffen
	if errors.Is(err, io.EOF) {
		connIoEOFError(socket)
	} else if errors.Is(err, syscall.ECONNRESET) {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
	} else if errors.Is(err, syscall.EPIPE) {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
	} else {
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
	}
	return true
}

// runningBackgroundServingLoop gibt an, ob die Hintergrund-Dauerschleifen eines Sockets aktiv sein sollen.
// Diese Funktion überprüft, ob die Verbindung des gegebenen BngConn-Objekts geschlossen wurde. Wenn die
// Verbindung nicht geschlossen ist, sollen die Hintergrund-Dauerschleifen weiterhin laufen.
//
// Parameter:
//   - ipcc *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und zugehörige
//     Ressourcen verwaltet.
//
// Rückgabe:
//   - bool: true, wenn die Hintergrund-Dauerschleifen weiterhin laufen sollen, ansonsten false.
func runningBackgroundServingLoop(ipcc *BngConn) bool {
	return !connectionIsClosed(ipcc)
}

// connectionIsClosed gibt an, ob eine Verbindung geschlossen wurde.
// Diese Funktion prüft verschiedene Zustände des BngConn-Objekts, um festzustellen, ob die Verbindung
// bereits geschlossen ist oder ein Fehler aufgetreten ist, der die Verbindung beendet hat.
//
// Parameter:
//   - ipcc *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und zugehörige
//     Ressourcen verwaltet.
//
// Rückgabe:
//   - bool: true, wenn die Verbindung geschlossen wurde, ansonsten false.
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

// fullCloseConn wird verwendet, um den Socket vollständig zu schließen.
// Diese Funktion sorgt dafür, dass die Verbindung ordnungsgemäß geschlossen wird, indem sie
// verschiedene Schritte durchführt, um sicherzustellen, dass alle Ressourcen freigegeben werden.
// Sie markiert die Verbindung als geschlossen, schließt den Socket, wartet auf den Abschluss aller
// Hintergrundprozesse und setzt das geschlossene Flag. Bei Fehlern während des Schließvorgangs
// wird der Fehler zurückgegeben.
//
// Parameter:
//   - s *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und zugehörige
//     Ressourcen verwaltet.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Schließen der Verbindung ein Problem aufgetreten ist, ansonsten nil.s
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
