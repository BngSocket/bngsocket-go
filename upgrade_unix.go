package bngsocket

import (
	"crypto/tls"
	"fmt"
	"net"

	"golang.org/x/net/websocket"
)

// UpgradeSocketToBngConn wandelt einen gegebenen net.Conn in ein *BngConn Objekt um.
// Diese Funktion prüft zunächst den Typ des übergebenen Sockets und erstellt basierend darauf
// ein neues BngConn Objekt. Unterstützte Verbindungstypen sind *net.UnixConn, *net.TCPConn,
// *websocket.Conn, *tls.Conn und allgemeine net.Conn. Nach der erfolgreichen Erstellung
// des BngConn Objekts werden die erforderlichen Hintergrundprozesse gestartet.
//
// Parameter:
//   - socket net.Conn: Das zu upgradende Socket, das verschiedene Verbindungstypen unterstützen kann.
//
// Rückgabe:
//   - *BngConn: Ein Zeiger auf das neu erstellte BngConn Objekt, das die Socket-Verbindung verwaltet.
//   - error: Ein Fehler, falls der Verbindungstyp nicht unterstützt wird oder ein anderer Fehler auftritt.
func UpgradeSocketToBngConn(socket net.Conn) (*BngConn, error) {
	// Es wird geprüft, ob es sich um einen zulässigen Socket handelt
	// Außerdem wird das Basis BNG Objekt erzeugt
	var client *BngConn
	switch socket.(type) {
	case *net.UnixConn:
		client = _NewBaseBngSocketObject(socket)
	case *net.TCPConn:
		client = _NewBaseBngSocketObject(socket)
	case *websocket.Conn:
		client = _NewBaseBngSocketObject(socket)
	case *tls.Conn:
		client = _NewBaseBngSocketObject(socket)
	default:
		return nil, ErrUnsupportedSocketType
	}

	// Die Anzahl der Routinen wird übermittelt
	client.backgroundProcesses.Add(2)

	// Debug-Ausgabe zur Bestätigung des Upgrades
	_DebugPrint(fmt.Sprintf("Connection upgraded to BngConn = %s", client._innerhid))

	// Es wird eine Routine gestartet, welche für das Senden der ausgehenden Daten ist
	//go constantWriting(client)

	// Es wird eine Routine gestartet, welche Parameter Daten liest
	go constantReading(client)

	// Das Objekt wird zurückgegeben
	return client, nil
}
