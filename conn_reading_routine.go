package bngsocket

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

// handleMessage liest und verarbeitet eine MSG-Nachricht aus dem Reader der BngConn.
// Die empfangenen Daten werden in den bereitgestellten Cache geschrieben. Die Funktion
// liest zunächst die Länge der Nachricht (Big-Endian), anschließend die eigentlichen
// Daten. Nach dem Schreiben in den Cache wird eine Checksumme der Daten berechnet und
// eine Debug-Ausgabe erzeugt.
//
// Parameter:
//   - cache *bytes.Buffer: Ein Puffer, in den die empfangenen Daten geschrieben werden.
//   - o *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und
//     zugehörige Ressourcen verwaltet.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Lesen oder Verarbeiten der Nachricht ein Problem
//     aufgetreten ist, ansonsten nil.s
func handleMessage(cache *bytes.Buffer, o *BngConn) error {
	// Lesen der Datenlänge (Big-Endian)
	var dataLength uint32
	if err := binary.Read(o.reader, binary.BigEndian, &dataLength); err != nil {
		return fmt.Errorf("%s: %w: %v", o._innerhid, ErrMessageLength, err)
	}

	// Lesen der Nachrichtendaten
	data := make([]byte, dataLength)
	if _, err := io.ReadFull(o.reader, data); err != nil {
		return fmt.Errorf("%s: %w: %v", o._innerhid, ErrMessageRead, err)
	}

	// Speichern der Daten im Cache
	cache.Write(data)

	// Berechne die Checksumme
	checksum := crc32.ChecksumIEEE(data)

	// Debug-Ausgabe: Checksumme und Länge der Nachricht
	_DebugPrint(fmt.Sprintf("BngConn(%s): MSG received, length=%d, checksum=%08x", o._innerhid, len(data), checksum))

	return nil
}

// handleEndTransfer verarbeitet das Ende eines Datentransfers (ET-Nachricht).
// Die Funktion berechnet die Checksumme der im Cache gespeicherten Daten und startet
// die Verarbeitung dieser Daten in einer separaten Goroutine. Nach der erfolgreichen
// Verarbeitung wird der Cache geleert.
//
// Parameter:
//   - o *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und
//     zugehörige Ressourcen verwaltet.
//   - cache *bytes.Buffer: Ein Puffer, der die empfangenen Daten enthält.
//
// Rückgabe:
//   - error: Ein Fehler, falls bei der Verarbeitung des Datensatzes ein Problem
//     aufgetreten ist, ansonsten nil.s
func handleEndTransfer(o *BngConn, cache *bytes.Buffer) error {
	// Berechne die Checksumme der Daten im Cache
	data := cache.Bytes()
	checksum := crc32.ChecksumIEEE(data)

	// Debug-Ausgabe: Länge und Checksumme der Daten
	_DebugPrint(fmt.Sprintf("BngConn(%s): ET received: Processing data (length=%d, checksum=%08x)", o._innerhid, len(data), checksum))

	// Starte die Verarbeitung in einer Goroutine
	o.backgroundProcesses.Add(1) // Informiere Wartungsgruppe
	go func(data []byte) {
		defer o.backgroundProcesses.Done() // Abschluss melden
		o._ProcessReadedData(data)         // Interne Verarbeitung
	}(data)

	// Cache leeren
	cache.Reset()

	// Erfolgreich verarbeitet
	return nil
}

// handleACK verarbeitet eine eingehende ACK-Nachricht.
// Die Funktion liest die ACK-Daten aus dem Reader der BngConn, prüft die
// Korrektheit der Nachricht und bestätigt das ACK durch Aufruf von EnterACK.
// Bei erfolgreicher Verarbeitung wird eine Debug-Ausgabe erzeugt.
//
// Parameter:
//   - o *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und
//     zugehörige Ressourcen verwaltet.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Lesen oder Verarbeiten der ACK ein Problem
//     aufgetreten ist, ansonsten nil.s
func handleACK(o *BngConn) error {
	ack := make([]byte, 2) // Das erste 'A' wurde schon gelesen
	if _, err := io.ReadFull(o.reader, ack); err != nil {
		return fmt.Errorf("%s: %w: %v", o._innerhid, ErrACKReadFailure, err)
	}

	// Prüfen, ob die Nachricht korrekt ist
	if string(ack) != "CK" {
		return fmt.Errorf("%s: %w", o._innerhid, ErrInvalidACK)
	}

	// Rufe die Methode `EnterACK` auf, um ACK zu bestätigen
	if err := o.ackHandle.EnterACK(); err != nil {
		return fmt.Errorf("%s: failed to process ACK: %v", o._innerhid, err)
	}

	_DebugPrint(fmt.Sprintf("BngConn(%s): ACK successfully processed", o._innerhid))
	return nil
}

// constantReading führt eine kontinuierliche Leseschleife auf der Socket-Verbindung des
// gegebenen BngConn-Objekts durch. Die Funktion verarbeitet eingehende Nachrichten basierend
// auf ihrem Typ:
//   - 'M' (MSG): Ein Teil des Datensatzes wird empfangen und verarbeitet. Nach erfolgreicher
//     Verarbeitung wird eine Bestätigung (ACK) zurückgesendet.
//   - 'E' (ET): Das Ende eines Datensatzes wird empfangen. Der gesamte Datensatz aus dem
//     Cache wird verarbeitet und eine Bestätigung (ACK) wird zurückgesendet.
//   - 'A' (ACK): Eine eingehende Bestätigung wird verarbeitet.
//
// Bei Auftreten von Fehlern während des Lese- oder Verarbeitungsprozesses wird die
// Funktion `readProcessErrorHandling` aufgerufen, um den Fehler zu behandeln. Abhängig von der
// Fehlerbehandlung kann die Verbindung geschlossen oder der Cache zurückgesetzt werden, um
// den Vorgang neu zu starten.
//
// Die Schleife läuft solange, wie die Funktion `runningBackgroundServingLoop(o)` den Wert
// `true` zurückgibt. Beim Start und beim Stoppen der Leseschleife werden Debug-Informationen
// ausgegeben. Nach Beendigung der Funktion wird die Hintergrundprozesszählung
// (`o.backgroundProcesses.Done()`) aufgerufen, um den Abschluss des Lesevorgangs zu signalisieren.
//
// Parameter:
//   - o *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und zugehörige
//     Ressourcen verwaltet.s
func constantReading(o *BngConn) {
	defer func() {
		_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was stopped", o._innerhid))
		o.backgroundProcesses.Done()
	}()

	_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was started", o._innerhid))

	var cache bytes.Buffer // Cache für MSG-Daten

	for runningBackgroundServingLoop(o) {
		// Lesen des Typ-Bytes
		msgType, err := o.reader.ReadByte()
		if err != nil {
			readProcessErrorHandling(o, err)
			break
		}

		switch msgType {
		case 'M': // MSG: Ein Teil des Datensatzes
			_DebugPrint(fmt.Sprintf("BngConn(%s): MSG received", o._innerhid))
			if err := handleMessage(&cache, o); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
			// Sende ACK zurück
			if err := writePacketACK(o); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
		case 'E': // ET: Ende des Datensatzes
			_DebugPrint(fmt.Sprintf("BngConn(%s): ET received", o._innerhid))
			// Jetzt den kompletten Datensatz aus dem Cache verarbeiten
			if err := handleEndTransfer(o, &cache); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
			// Sende ACK zurück
			if err := writePacketACK(o); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
		case 'A': // ACK: Eingehende Bestätigung
			_DebugPrint(fmt.Sprintf("BngConn(%s): ACK received", o._innerhid))
			if err := handleACK(o); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
		default:
			// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
			if readProcessErrorHandling(o, ErrUnknownMessageType) {
				return
			} else {
				// Der Cache wird geelert und der Vorgang wird zurückgesetzt
				cache.Reset()
				continue
			}
		}
	}
}
