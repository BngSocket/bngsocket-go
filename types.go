package bngsocket

import (
	"bufio"
	"bytes"
	"net"
	"reflect"
	"sync"

	"github.com/CustodiaJS/bngsocket/transport"
)

// _DataItem repräsentiert einen einzelnen Datensatz mit einer eindeutigen ID und den zugehörigen Daten.
type _DataItem struct {
	id        uint64        // Eindeutige ID des Datensatzes
	data      *bytes.Buffer // Puffer, der die Daten des Datensatzes enthält
	totalSize int           // Gesamtgröße des Datensatzes in Bytes
	bytesRead int           // Anzahl der bereits gelesenen Bytes
}

// _ByteCache speichert mehrere Datensätze und ermöglicht das sequenzielle Lesen dieser Daten.
type _ByteCache struct {
	dataItems []*_DataItem // Liste von Datensätzen, die im Cache gespeichert sind
	closed    bool         // Gibt an, ob das Objekt geschlossen wurde
	currentID uint64       // ID für den nächsten hinzuzufügenden Datensatz
	mu        sync.Mutex   // Mutex für die Synchronisation beim Zugriff auf die Daten
	cond      *sync.Cond   // Bedingungsvariable, um auf das Vorhandensein von Daten zu warten
}

// _ConnACK verwaltet die Zustände für ACK-Nachrichten in der Verbindung.
type _ConnACK struct {
	cond  *sync.Cond  // Bedingungsvariable für ACK-Zustandsänderungen
	mutex *sync.Mutex // Mutex zum Schutz des Zustands
	state uint8       // Aktueller Zustand der ACK-Verarbeitung
}

// BngConn stellt die Verbindung und den Status eines BNG (Broadband Network Gateway) dar.
type BngConn struct {
	_innerhid string // Eindeutige interne ID der Verbindung

	// Verbindung und I/O
	conn      net.Conn      // Socket-Verbindung des BNG
	writer    *bufio.Writer // Buffered Writer für die Verbindung
	reader    *bufio.Reader // Buffered Reader für die Verbindung
	ackHandle *_ConnACK     // Verwaltung der ACK-Nachrichten
	connMutex *sync.Mutex   // Mutex zum Schutz der Verbindung

	// Sitzungszustand
	closed       _SafeBool         // Flag, das angibt, ob der Socket geschlossen wurde
	closing      _SafeBool         // Flag, das angibt, ob der Socket geschlossen werden soll
	runningError _SafeValue[error] // Speichert Fehler, die während des Betriebs auftreten

	// Writer-Synchronisation
	writerMutex *sync.Mutex // Mutex zum Schutz des Writers

	// RPC-Variablen
	functions           _SafeMap[string, reflect.Value]               // Registrierte Funktionen
	openRpcRequests     _SafeMap[string, chan *transport.RpcResponse] // Offene RPC-Anfragen
	backgroundProcesses *sync.WaitGroup                               // Wartet auf laufende Hintergrundprozesse

	// Channel-Variablen
	openChannelListener      _SafeMap[string, *BngConnChannelListener]                // Verfügbare Channel-Listener
	openChannelInstances     _SafeMap[string, *BngConnChannel]                        // Aktive Channel-Instanzen
	openChannelJoinProcesses _SafeMap[string, chan *transport.ChannelRequestResponse] // Offene Channel-Join-Prozesse
}

// BngRequest stellt eine Anfrage an eine BNG-Verbindung dar.
type BngRequest struct {
	Conn *BngConn // Verweis auf die BNG-Verbindung, die diese Anfrage bearbeitet
}

// bngConnAcceptingRequest beschreibt eine Anfrage, um einen neuen Channel zu akzeptieren.
type bngConnAcceptingRequest struct {
	requestedChannelId string // ID des angeforderten Channels
	requestChannelid   string // ID des Channels, über den die Anfrage akzeptiert wird
}

// BngConnChannelListener hört auf eingehende Verbindungen für einen spezifischen BNG-Channel.
type BngConnChannelListener struct {
	mu              *sync.Mutex                          // Mutex für den Zugriffsschutz auf den Listener
	socket          *BngConn                             // Verweis auf die BNG-Verbindung
	waitOfAccepting *_SafeChan[*bngConnAcceptingRequest] // Kanal für akzeptierte Channel-Anfragen
}

// BngConnChannel repräsentiert einen Channel für die Kommunikation über eine BNG-Verbindung.
type BngConnChannel struct {
	socket              *BngConn          // BNG-Verbindung, über die dieser Channel läuft
	sesisonId           string            // Aktuelle Session-ID für den Channel
	isClosed            _SafeBool         // Flag, das angibt, ob der Channel geschlossen wurde
	waitOfPackageACK    _SafeBool         // Flag, das angibt, ob auf ein ACK-Paket gewartet wird
	currentReadingCache _SafeBytes        // Cache für Daten, die gerade gelesen werden
	openReaders         _SafeInt          // Zähler für die Anzahl der aktuell offenen Leseoperationen
	openWriters         _SafeInt          // Zähler für die Anzahl der aktuell offenen Schreiboperationen
	bytesDataInCache    *_ByteCache       // Cache für die eingehenden Daten
	ackChan             _SafeAck          // Kanal für ACK-Rückmeldungen
	channelRunningError _SafeValue[error] // Speichert Fehler ab, welche bei der Verwendung des Channels auftreten können
	mu                  *sync.Mutex       // Mutex zum Schutz des Channels
}
