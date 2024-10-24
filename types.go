package bngsocket

import (
	"bytes"
	"net"
	"reflect"
	"sync"
)

// writingState speichert den aktuellen Schreibstatus, einschließlich der Anzahl der geschriebenen Bytes und eines möglichen Fehlers.
type writingState struct {
	n   int   // Anzahl der erfolgreich geschriebenen Bytes
	err error // Fehler, der während des Schreibvorgangs aufgetreten ist
}

// dataWritingResolver hält die zu schreibenden Daten und einen Kanal für die Statusrückmeldung.
type dataWritingResolver struct {
	data          []byte             // Die zu schreibenden Daten
	waitOfResolve chan *writingState // Kanal für die Rückmeldung des Schreibstatus
}

// DataItem repräsentiert einen einzelnen Datensatz mit einer eindeutigen ID und den zugehörigen Daten.
type _DataItem struct {
	id        uint64        // Eindeutige ID des Datensatzes
	data      *bytes.Buffer // Puffer, der die Daten des Datensatzes enthält
	totalSize int           // Gesamtgröße des Datensatzes in Bytes
	bytesRead int           // Anzahl der bereits gelesenen Bytes
}

// ByteCache speichert mehrere Datensätze und ermöglicht das sequenzielle Lesen dieser Daten.
type ByteCache struct {
	dataItems []*_DataItem // Liste von Datensätzen, die im Cache gespeichert sind
	closed    bool         // Gibt an ob das Objekt geschlossen wurde
	currentID uint64       // ID für den nächsten hinzuzufügenden Datensatz
	mu        sync.Mutex   // Mutex für die Synchronisation beim Zugriff auf die Daten
	cond      *sync.Cond   // Bedingungsvariable, um auf das Vorhandensein von Daten zu warten
}

// BngConn stellt die Verbindung und den Status eines BNG (Broadband Network Gateway) dar.
type BngConn struct {
	_innerhid                string
	mu                       *sync.Mutex                                   // Mutex für den allgemeinen Zugriffsschutz
	conn                     net.Conn                                      // Socket-Verbindung des BNG
	closed                   SafeBool                                      // Flag, das angibt, ob der Socket geschlossen wurde
	closing                  SafeBool                                      // Flag, das angibt, ob der Socket geschlossen werden soll
	functions                SafeMap[string, reflect.Value]                // Speichert die registrierten Funktionen
	runningError             SafeValue[error]                              // Speichert Fehler, die während des Betriebs auftreten
	writingChan              *SafeChan[*dataWritingResolver]               // Kanal für sendbare Daten
	openRpcRequests          SafeMap[string, chan *RpcResponse]            // Speichert alle offenen RPC-Anfragen
	hiddenFunctions          SafeMap[string, reflect.Value]                // Speichert die versteckten (geteilten) Funktionen
	backgroundProcesses      *sync.WaitGroup                               // Wartet auf laufende Hintergrundprozesse
	openChannelListener      SafeMap[string, *BngConnChannelListener]      // Speichert alle verfügbaren Channel-Listener
	openChannelInstances     SafeMap[string, *BngConnChannel]              // Speichert alle aktiven Channel-Instanzen
	openChannelJoinProcesses SafeMap[string, chan *ChannelRequestResponse] // Speichert alle offenen Channel-Join-Prozesse
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
	mu              *sync.Mutex                         // Mutex für den Zugriffsschutz auf den Listener
	socket          *BngConn                            // Verweis auf die BNG-Verbindung
	waitOfAccepting *SafeChan[*bngConnAcceptingRequest] // Kanal für akzeptierte Channel-Anfragen
}

// BngConnChannel repräsentiert einen Channel für die Kommunikation über eine BNG-Verbindung.
type BngConnChannel struct {
	socket              *BngConn         // Speichert die BNG-Verbindung, über die dieser Channel läuft
	sesisonId           string           // Aktuelle Session-ID für den Channel
	isClosed            SafeBool         // Flag, das angibt, ob der Channel geschlossen wurde
	waitOfPackageACK    SafeBool         // Flag, das angibt, ob auf ein ACK-Paket gewartet wird
	currentReadingCache SafeBytes        // Cache für Daten, die gerade gelesen werden
	openReaders         SafeInt          // Zähler für die Anzahl der aktuell offenen Leseoperationen
	openWriters         SafeInt          // Zähler für die Anzahl der aktuell offenen Schreiboperationen
	bytesDataInCache    *ByteCache       // Cache für die eingehenden Daten
	ackChan             SafeAck          // Kanal für ACK-Rückmeldungen
	channelRunningError SafeValue[error] // Speichert Fehler ab, welche bei der Verwendung des Channels auftreten können
	mu                  *sync.Mutex      // Objekt Mutex
}
