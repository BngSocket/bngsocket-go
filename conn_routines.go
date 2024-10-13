package bngsocket

import (
	"bufio"
	"encoding/binary"
	"fmt"
)

// Ließt Chunks ein und verarbeitet sie weiter
func constantReading(o *BngConn) {
	// Wenn die Funktion am ende ist wird Signalisiert dass sie zuennde ist
	defer o.bp.Done()

	// Debug am ende
	defer DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was stopped", o._innerhid))

	// Der Buffer Reader ließt die Daten und speichert sie dann im Cache
	reader := bufio.NewReader(o.conn)
	cacheBytes := make([]byte, 0)
	cachedData := make([]byte, 0)

	// Debug
	DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was started", o._innerhid))

	// Diese Schleife wird Permanent ausgeführt und liest alle Daten ein
	for RunningBackgroundServingLoop(o) {
		// Verfügbare Daten aus der Verbindung lesen
		rBytes := make([]byte, 4096)
		sizeN, err := reader.Read(rBytes)
		if err != nil {
			// Der Fehler wird ausgewertet
			ReadProcessErrorHandling(o, err)
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
					ReadProcessErrorHandling(o, fmt.Errorf("chunk too large: %d bytes", length))
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
				o.bp.Add(1)

				// Debug
				DebugPrint(fmt.Sprintf("BngConn(%s): %d bytes was recived", o._innerhid, len(transportBytes)+ics))

				// Die Daten werden durch die GoRoutine verarbeitet
				go func(data []byte) {
					defer o.bp.Done()
					o._ProcessReadedData(data)
				}(transportBytes)
			} else {
				// Der Fehler wird ausgewertet
				ReadProcessErrorHandling(o, fmt.Errorf("unknown message type: %v", cacheBytes[0]))
				return
			}
		}
	}
}

// Schreibt kontinuirlich Chunks sobald verfügbar
func constantWriting(o *BngConn) {
	// Wird am ende der Lesefunktion aufgerufen
	defer o.bp.Done()

	// Debug am ende
	defer DebugPrint(fmt.Sprintf("BngConn(%s): Constant writing to Socket was stopped", o._innerhid))

	// Der Writer wird erstellt
	writer := bufio.NewWriter(o.conn)

	// Debug
	DebugPrint(fmt.Sprintf("BngConn(%s): Constant writing to Socket was started", o._innerhid))

	// Die Schleife empfängt die Daten
	for RunningBackgroundServingLoop(o) {
		// Es wird auf neue Daten aus dem Chan gewartet
		data, ok := o.writingChan.Read()
		if !ok {
			// Es wird geprüft ob die Verbindung getrennt wurde
			if ConnectionIsClosed(o) {
				break
			}

			// Es handelt sich um einen Schwerwiegenden Fehler, die Verbindung wird getrennt
			o._ConsensusProtocolTermination(fmt.Errorf("internal chan error"))

			// Schleife, abbruch
			break
		}

		// Daten in Chunks aufteilen
		chunks := SplitDataIntoChunks(data.data, 4096)

		// Die Chunks werden übertragen
		writedBytes := uint64(0)
		for _, chunk := range chunks {
			// Es wird geprüft ob die Verbindung getrennt wurde
			if ConnectionIsClosed(o) {
				// Der Fehler wird verarbeitet
				o._ConsensusConnectionClosedSignal()
				break
			}

			// Nachrichtentyp 'C' senden
			err := writer.WriteByte('C')
			if err != nil {
				// Der Fehler wird verarbeitet
				WriteProcessErrorHandling(o, err)
				break
			}
			writedBytes = writedBytes + 1

			// Chunk-Länge senden (2 Bytes)
			length := uint16(len(chunk))
			lengthBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(lengthBytes, length)
			_, err = writer.Write(lengthBytes)
			if err != nil {
				// Der Fehler wird verarbeitet
				WriteProcessErrorHandling(o, err)
				break
			}
			writedBytes = writedBytes + uint64(len(chunk))

			// Chunk-Daten senden
			_, err = writer.Write(chunk)
			if err != nil {
				// Der Fehler wird verarbeitet
				WriteProcessErrorHandling(o, err)
				break
			}

			// Flush, um sicherzustellen, dass die Daten gesendet werden
			err = writer.Flush()
			if err != nil {
				// Der Fehler wird verarbeitet
				WriteProcessErrorHandling(o, err)
				break
			}
		}

		// Es wird geprüft ob die Verbindung getrennt wurde
		if ConnectionIsClosed(o) {
			// Der Fehler wird verarbeitet
			o._ConsensusConnectionClosedSignal()
			break
		}

		// Nachrichtentyp 'L' senden, um das Ende der Nachricht zu signalisieren
		err := writer.WriteByte('L')
		if err != nil {
			// Der Fehler wird verarbeitet
			WriteProcessErrorHandling(o, err)
			break
		}
		writedBytes = writedBytes + 1

		// Die Übertragung wird fertigestellt
		err = writer.Flush()
		if err != nil {
			// Der Fehler wird verarbeitet
			WriteProcessErrorHandling(o, err)
			break
		}

		// Debug
		DebugPrint(fmt.Sprintf("BngConn(%s): %d bytes writed", o._innerhid, writedBytes))

		// Es wird Signalisiert dass die Übertragung erfolgreich war
		data.waitOfResolve <- &writingState{n: int(int(writedBytes) - 1 - len(chunks)), err: nil}
	}
}
