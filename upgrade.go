package bngsocket

import (
	"net"
	"reflect"
	"sync"
)

// Wird verwendet um Vorhandene Sockets zu einem BngSocket zu verwandeln
func UpgradeSocketToBngSocket(socket net.Conn) (*BngSocket, error) {
	// Das Client Objekt wird erstellt
	client := &BngSocket{
		bp:              &sync.WaitGroup{},
		mu:              &sync.Mutex{},
		conn:            socket,
		closed:          false,
		writeableData:   make(chan []byte),
		functions:       make(map[string]reflect.Value),
		openRpcRequests: make(map[string]chan *RpcResponse),
		hiddenFunctions: make(map[string]reflect.Value),
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
