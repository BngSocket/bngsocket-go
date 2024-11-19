func constantReading(o *BngConn, wg *sync.WaitGroup) {
	defer func() {
		_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was stopped", o._innerhid))
		wg.Done() // Signalisiert, dass die Leseschleife beendet ist
	}()

	cache := make([]byte, 0) // Zwischenspeicher für empfangene Daten
	buffer := make([]byte, 4096)

	_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was started", o._innerhid))

	for {
		// Daten vom Socket lesen
		sizeN, err := o.conn.Read(buffer)
		if err != nil {
			_DebugPrint(fmt.Sprintf("BngConn(%s): Connection error: %v", o._innerhid, err))
			o.conn.Close() // Verbindung schließen bei Fehler
			return
		}

		// Wenn 0 Bytes empfangen werden, Verbindung schließen
		if sizeN == 0 {
			_DebugPrint(fmt.Sprintf("BngConn(%s): Connection closed by peer", o._innerhid))
			o.conn.Close() // Verbindung sauber schließen
			return
		}

		// Füge empfangene Daten in den Cache ein
		cache = append(cache, buffer[:sizeN]...)

		// **Sofortiges ACK für empfangene Teilpakete**
		err = sendACK(o.conn, 0)
		if err != nil {
			_DebugPrint(fmt.Sprintf("BngConn(%s): Error sending ACK for part: %v", o._innerhid, err))
			o.conn.Close() // Verbindung schließen bei ACK-Fehler
			return
		}

		// Verarbeiten der vollständigen Pakete im Cache
		for len(cache) > 0 { // Verarbeiten, solange Daten im Cache sind
			messageType := cache[0]
			switch messageType {
			case 2: // Datenpaket (CHUNK)
				if len(cache) < 3 { // Header unvollständig
					break // Warten auf mehr Daten
				}

				// Länge des Pakets aus dem Header lesen
				packetLength := int(binary.BigEndian.Uint16(cache[1:3]))
				totalLength := 3 + packetLength // Nachrichtentyp (1) + Länge (2) + Daten

				if len(cache) < totalLength { // Paket unvollständig
					break // Warten auf mehr Daten
				}

				// Paketdaten extrahieren und verarbeiten
				packetData := cache[3:totalLength]
				_DebugPrint(fmt.Sprintf("BngConn(%s): Processing packet of size %d", o._innerhid, len(packetData)))

				// Paketdaten verarbeiten (z. B. weiterleiten oder speichern)
				processCompleteData(o, packetData)

				// ACK senden, um den Abschluss des Pakets zu bestätigen
				err := sendACK(o.conn, 0)
				if err != nil {
					_DebugPrint(fmt.Sprintf("BngConn(%s): Error sending ACK for packet: %v", o._innerhid, err))
					o.conn.Close() // Verbindung schließen bei ACK-Fehler
					return
				}

				// Paketdaten aus dem Cache entfernen
				cache = cache[totalLength:]

			case 3: // Abschlussnachricht
				if len(cache) < 7 { // Abschlussnachricht unvollständig
					break // Warten auf mehr Daten
				}

				// CRC32 der empfangenen Daten validieren
				receivedCRC := binary.BigEndian.Uint32(cache[1:5])
				cache = cache[5:] // Typ und CRC entfernen

				// Berechnung der CRC32-Prüfsumme
				calculatedCRC := crc32.ChecksumIEEE(cache)

				if calculatedCRC != receivedCRC {
					_DebugPrint(fmt.Sprintf("BngConn(%s): CRC mismatch", o._innerhid))
					err := sendACK(o.conn, 1) // NACK senden bei Fehler
					if err != nil {
						readProcessErrorHandling(o, err)
						return
					}
					continue
				}

				// Abschluss-ACK senden
				_DebugPrint(fmt.Sprintf("BngConn(%s): Final ACK sent", o._innerhid))
				err = sendACK(o.conn, 0) // Abschluss-ACK senden
				if err != nil {
					readProcessErrorHandling(o, err)
					return
				}

			default: // Unbekannter Nachrichtentyp
				_DebugPrint(fmt.Sprintf("BngConn(%s): Unknown message type %d, skipping", o._innerhid, messageType))
				cache = cache[1:] // Unbekannte Nachricht entfernen
			}
		}
	}
}
