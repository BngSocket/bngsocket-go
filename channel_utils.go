package bngsocket

import (
	"io"
)

// _IsClosedOrHasRunningErrorOnChannel prüft den Status eines BngConnChannel.
// Die Funktion gibt zurück, ob der Channel geschlossen wurde, ob die Hauptverbindung geschlossen wurde und ob ein laufender Fehler vorliegt.
//
// Parameter:
//   - channel *BngConnChannel: Ein Zeiger auf das BngConnChannel-Objekt, dessen Status überprüft werden soll.
//
// Rückgabe:
//   - channelWasClosed bool: Gibt an, ob der Channel geschlossen wurde.
//   - connWasClosed bool: Gibt an, ob die Hauptverbindung geschlossen wurde.
//   - runningError error: Gibt einen laufenden Fehler zurück, falls vorhanden, ansonsten nil.
func _IsClosedOrHasRunningErrorOnChannel(channel *BngConnChannel) (channelWasClosed bool, connWasClosed bool, runningError error) {
	// Es wird geprüft ob ein Fehler vorhanden ist
	if err := channel.channelRunningError.Get(); err != nil {
		return true, false, err
	}

	// Es wird geprüft ob die Verbindung geschlossen wurde
	if channel.isClosed.Get() {
		return true, false, io.EOF
	}

	// Es wird geprüft ob die Hauptverbindung geschlossen ist
	if channel.socket.closed.Get() || channel.socket.closing.Get() {
		return false, true, io.EOF
	}

	// Der Channel ist geöffnet und es ist kein Fehler vorhanden
	return false, false, nil
}
