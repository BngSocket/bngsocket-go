package bngsocket

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

// Ließt Chunks ein und verarbeitet sie weiter
func constantReadingLagacy(o *BngConn) {
	defer func() {
		// DEBUG Log
		_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was stopped", o._innerhid))

		// Es wird eine Hintergrundaufgabe für erledigt Markeirt
		o.backgroundProcesses.Done()
	}()

	// Der Buffer Reader ließt die Daten und speichert sie dann im Cache
	reader := bufio.NewReader(o.conn)
	cacheBytes := make([]byte, 0)
	cachedData := make([]byte, 0)

	// Debug
	_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was started", o._innerhid))

	// Diese Schleife wird Permanent ausgeführt und liest alle Daten ein
	for runningBackgroundServingLoop(o) {
		// Verfügbare Daten aus der Verbindung lesen
		rBytes := make([]byte, 4096)
		sizeN, err := reader.Read(rBytes)
		if err != nil {
			readProcessErrorHandling(o, err)
			return
		}

		// Die Daten werden im Cache zwischengespeichert
		cacheBytes = append(cacheBytes, rBytes[:sizeN]...)

		// Verarbeitung des Empfangspuffers
		ics := 0
		for {
			// Es wird geprüft ob Mindestens 1 Byte im Cache ist
			if len(cacheBytes) < 1 {
				break // Nicht genug Daten, um den Nachrichtentyp zu bestimmen
			}

			// Der Datentyp wird ermittelt
			if cacheBytes[0] == byte('C') {
				ics = ics + 1

				// Prüfen, ob genügend Daten für die Chunk-Länge vorhanden sind
				if len(cacheBytes) < 3 { // 1 Byte für 'C' und 2 Bytes für die Länge
					break // Warten auf mehr Daten
				}

				// Chunk-Länge lesen (Bytes 1 bis 2)
				length := binary.BigEndian.Uint16(cacheBytes[1:3])

				// Optional: Maximale erlaubte Chunk-Größe überprüfen
				const MaxChunkSize = 4096 // Maximal erlaubte Chunk-Größe in Bytes
				if length > MaxChunkSize {
					// Der Fehler wird ausgewertet
					readProcessErrorHandling(o, fmt.Errorf("chunk too large: %d bytes", length))
					break
				}

				// Prüfen, ob genügend Daten für den gesamten Chunk vorhanden sind
				totalLength := 1 + 2 + int(length) // 'C' + Länge (2 Bytes) + Chunk-Daten
				if len(cacheBytes) < totalLength {
					break // Warten auf mehr Daten
				}

				// Chunk-Daten extrahieren
				chunkData := cacheBytes[3:totalLength]

				// Chunk-Daten dem gecachten Daten hinzufügen
				cachedData = append(cachedData, chunkData...)

				// Verarbeitete Bytes aus dem Cache entfernen
				cacheBytes = cacheBytes[totalLength:]
			} else if cacheBytes[0] == byte('L') {
				// 'L' aus dem Cache entfernen
				cacheBytes = cacheBytes[1:]
				ics = ics + 1

				// Gesammelte Daten verarbeiten
				transportBytes := make([]byte, len(cachedData))
				copy(transportBytes, cachedData)
				cachedData = make([]byte, 0)
				o.backgroundProcesses.Add(1)

				// Debug
				_DebugPrint(fmt.Sprintf("BngConn(%s): %d bytes was recived", o._innerhid, len(transportBytes)+ics))

				// Die Daten werden durch die GoRoutine verarbeitet
				go func(data []byte) {
					defer o.backgroundProcesses.Done()
					o._ProcessReadedData(data)
				}(transportBytes)
			} else {
				// Der Fehler wird ausgewertet
				readProcessErrorHandling(o, fmt.Errorf("unknown message type: %v", cacheBytes[0]))
				break
			}
		}
	}
}

func constantReading(o *BngConn) {
	cache := make([]byte, 0)        // Zwischenspeicher für empfangene Daten
	completeData := make([]byte, 0) // Gesamte empfangene Daten
	buffer := make([]byte, 4096)    // Lesepuffer bleibt bei 4096 Bytes

	_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was started", o._innerhid))

	for runningBackgroundServingLoop(o) {
		// Direkt vom Socket lesen
		sizeN, err := o.conn.Read(buffer)
		if err != nil {
			readProcessErrorHandling(o, err)
			return
		}

		// An Cache anhängen
		cache = append(cache, buffer[:sizeN]...)

		// Verarbeiten aller vollständigen Nachrichten im Cache
		for {
			if len(cache) < 3 { // Mindestens 3 Bytes für Header erforderlich
				break
			}

			messageType := cache[0] // Nachrichtentyp bestimmen (0, 1, 2 oder 3)
			switch messageType {
			case 0: // ACK
				_DebugPrint(fmt.Sprintf("BngConn(%s): Received ACK (0)", o._innerhid))
				cache = cache[1:] // ACK aus dem Cache entfernen

				// Status setzen und Waiting Group signalisieren
				o.writerMutex.Lock()
				o.ackStatus = 0    // ACK gesetzt
				o.ackCond.Signal() // Signal an die Waiting Group
				o.writerMutex.Unlock()
			case 1: // NACK
				_DebugPrint(fmt.Sprintf("BngConn(%s): Received NACK (1)", o._innerhid))
				cache = cache[1:] // NACK aus dem Cache entfernen

				// Status setzen und Waiting Group signalisieren
				o.writerMutex.Lock()
				o.ackStatus = 1    // NACK gesetzt
				o.ackCond.Signal() // Signal an die Waiting Group
				o.writerMutex.Unlock()
			case 2: // CHUNK
				// Länge des Chunks ermitteln
				chunkLength := int(binary.BigEndian.Uint16(cache[1:3]))
				totalLength := 3 + chunkLength // Typ (1) + Länge (2) + Daten

				// Prüfen, ob der gesamte Chunk im Cache ist
				if len(cache) < totalLength {
					break // Auf mehr Daten warten
				}

				// Chunk-Daten extrahieren und anhängen
				chunkData := cache[3:totalLength]
				completeData = append(completeData, chunkData...)

				// ACK senden
				err := sendACK(o.conn, 0) // 0 = ACK
				if err != nil {
					readProcessErrorHandling(o, err)
					return
				}

				// Verarbeitete Bytes aus dem Cache entfernen
				cache = cache[totalLength:]
			case 3: // ABSCHLUSS
				if len(cache) < 7 { // Typ (1) + CRC32 (4) = 5 Bytes erforderlich
					break // Auf mehr Daten warten
				}

				// CRC32-Prüfsumme aus der Nachricht extrahieren
				receivedCRC := binary.BigEndian.Uint32(cache[1:5])
				cache = cache[5:] // Typ (1) und CRC32 (4) entfernen

				// Berechnung der CRC32-Prüfsumme der gesammelten Daten
				calculatedCRC := crc32.ChecksumIEEE(completeData)

				// Vergleich der CRC32-Prüfsummen
				if calculatedCRC != receivedCRC {
					readProcessErrorHandling(o, fmt.Errorf("CRC mismatch: expected %08x, got %08x", calculatedCRC, receivedCRC))
					sendACK(o.conn, 1) // 1 = NACK
					continue
				}

				// Gesammelte Daten final verarbeiten
				err := processCompleteData(o, completeData)
				if err != nil {
					readProcessErrorHandling(o, err)
					sendACK(o.conn, 1) // 1 = NACK
					return
				}

				// Abschluss-ACK senden
				err = sendACK(o.conn, 0) // 0 = ACK für Abschlussnachricht
				if err != nil {
					readProcessErrorHandling(o, err)
					return
				}

				// Datenpuffer leeren
				completeData = make([]byte, 0)
			default: // Unbekannter Nachrichtentyp
				readProcessErrorHandling(o, fmt.Errorf("unknown message type: %v", messageType))
				sendACK(o.conn, 1) // 1 = NACK
				cache = cache[1:]  // Unbekanntes Byte entfernen
			}
		}
	}
}
