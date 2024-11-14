package bngsocket

import (
	"crypto/tls"
	"fmt"
	"net"

	"golang.org/x/net/websocket"
)

// Wird verwendet um Vorhandene Sockets zu einem bngsocket zu verwandeln
func UpgradeSocketToBngConn(socket net.Conn) (*BngConn, error) {
	// Es wird geprüft ob es sich um einen Zulässigen Socket handelt
	// außerdem wird das Basis BNG Objekt erzeugt
	var client *BngConn
	switch cconn := socket.(type) {
	case *net.UnixConn:
		client = _NewBaseBngSocketObject(socket)
	case *net.TCPConn:
		client = _NewBaseBngSocketObject(socket)
	case *websocket.Conn:
		client = _NewBaseBngSocketObject(socket)
	case *tls.Conn:
		client = _NewBaseBngSocketObject(socket)
	case net.Conn:
		client = _NewBaseBngSocketObject(socket)
	default:
		return nil, fmt.Errorf("not supported connect type: %s", cconn)
	}

	// Die Anzahl der Routinen wird übermittelt
	client.backgroundProcesses.Add(2)

	// Debug
	_DebugPrint(fmt.Sprintf("Connection upraged to BngConn = %s", client._innerhid))

	// Es wird eine Routine gestartet welche für das Senden der Ausgehenden Daten ist
	//go constantWriting(client)

	// Wird eine Routine gestatet welche Parament Daten ließt
	go constantReading(client)

	// Das Objekt wird zurückgegeben
	return client, nil
}
