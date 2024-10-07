package bngsocket

import (
	"crypto/tls"
	"fmt"
	"net"

	"golang.org/x/net/websocket"
)

// Wird verwendet um Vorhandene Sockets zu einem bngsocket zu verwandeln
func UpgradeSocketToBngSocket(socket net.Conn) (*BngConn, error) {
	// Es wird geprüft ob es sich um einen Zulässigen Socket handelt
	// außerdem wird das Basis BNG Objekt erzeugt
	client := newBaseBngSocketObject()
	switch cconn := socket.(type) {
	case *net.UnixConn:
		client.conn = cconn
	case *net.TCPConn:
		client.conn = cconn
	case *websocket.Conn:
		client.conn = cconn
	case *tls.Conn:
		client.conn = cconn
	default:
		return nil, fmt.Errorf("not supported connect type")
	}

	// Die Anzahl der Routinen wird übermittelt
	client.bp.Add(2)

	// Es wird eine Routine gestartet welche für das Senden der Ausgehenden Daten ist
	go constantWriting(client)

	// Wird eine Routine gestatet welche Parament Daten ließt
	go constantReading(client)

	// Das Objekt wird zurückgegeben
	return client, nil
}
