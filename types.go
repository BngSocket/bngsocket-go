package bngsocket

import (
	"net"
	"reflect"
	"sync"
)

// BngSocket-Struktur
type BngSocket struct {
	bp              *sync.WaitGroup              // Gibt die Waitgroup für Offenene Hintergrund Prozesse an
	mu              *sync.Mutex                  // Gibt den Allgemeinen Mutex an
	conn            net.Conn                     // Gibt die Socket Verbindung an
	closed          bool                         // Gibt an ob der Socket geschlossen wurde
	closing         bool                         // Gibt an ob der Socket geschlossen werden soll
	runningError    error                        // Speichert Fehler ab, welche im Hintergrund auftreten
	writeableData   chan []byte                  // Übergibt Sendbare Daten
	openRpcRequests map[string]chan *RpcResponse // Speichert alle Offenenen Request zwischen
	functions       map[string]reflect.Value     // Speichert alle Funtionen ab
	hiddenFunctions map[string]reflect.Value     // Speichert alle Geteilten Funktionen (Hidden) an
}

// Context-Struktur, die zusätzliche Informationen während des Funktionsaufrufs hält
type BngRequest struct {
	Conn *BngSocket // Beispiel: WebSocket-Verbindung
}

type RingBuffer struct {
	data []byte
	head int
	tail int
	size int
}
