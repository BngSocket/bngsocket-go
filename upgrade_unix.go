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
	client := _NewBaseBngSocketObject()
	switch cconn := socket.(type) {
	case *net.UnixConn:
		client.conn = cconn
	case *net.TCPConn:
		client.conn = cconn
	case *websocket.Conn:
		client.conn = cconn
	case *tls.Conn:
		client.conn = cconn
	case net.Conn:
		client.conn = cconn
	default:
		return nil, fmt.Errorf("not supported connect type: %s", cconn)
	}

	// Die Anzahl der Routinen wird übermittelt
	client.backgroundProcesses.Add(2)

	// Debug
	DebugPrint(fmt.Sprintf("Connection upraged to BngConn = %s", client._innerhid))

	// Es wird eine Routine gestartet welche für das Senden der Ausgehenden Daten ist
	go constantWriting(client)

	// Wird eine Routine gestatet welche Parament Daten ließt
	go constantReading(client)

	// Das Objekt wird zurückgegeben
	return client, nil
}
