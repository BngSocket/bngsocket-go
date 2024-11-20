package bngsocket

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

func processCompleteData(o *BngConn, data []byte) error {
	// Startet eine neue Goroutine, um die Daten asynchron zu verarbeiten
	o.backgroundProcesses.Add(1) // Wartungsgruppe informieren
	go func(data []byte) {
		defer o.backgroundProcesses.Done() // Abschluss der Verarbeitung melden
		o._ProcessReadedData(data)         // Interne Verarbeitung aufrufen
	}(data)

	// Erfolgreich verarbeitet (die eigentliche Verarbeitung passiert in der Goroutine)
	return nil
}

func constantReading(o *BngConn) {
	defer func() {
		_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was stopped", o._innerhid))
		o.backgroundProcesses.Done()
	}()

	cache := make([]byte, 0) // Zwischenspeicher für empfangene Daten
	_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was started", o._innerhid))

	for runningBackgroundServingLoop(o) {
		// Die Daten aus dem Paket werden eingelesen
		buffer := make([]byte, 4096)
		sizeN, err := o.conn.Read(buffer)
		if err != nil {
			readProcessErrorHandling(o, err)
			return
		}

		// Es wird geprüft ob mindesttens 1 Zeichen auf dem Bzffer liegen
		if sizeN < 1 {
			readProcessErrorHandling(o, fmt.Errorf("invalid data stream"))
			return
		}

		switch {
		case buffer[0] == byte(3):
			_DebugPrint(fmt.Sprintf("BngConn(%s): flush recived", o._innerhid))

			// Es wird geprüft ob genügend Daten im Cache sind
			if len(cache) < 5 {
				readProcessErrorHandling(o, fmt.Errorf("cache too small for CRC processing"))
				return
			}

			// Es wird eine CR32 aus dem Cache berechnet
			dataWithoutCRC := cache[:len(cache)-4]
			calculatedCRC := crc32.ChecksumIEEE(dataWithoutCRC)

			// CRC32 in Bytes umwandeln
			crcBytes := make([]byte, 4) // CRC32 ist 4 Bytes groß
			binary.BigEndian.PutUint32(crcBytes, calculatedCRC)

			// Die CRC wird mit der CRC am ende des Paketes überprüft
			if !bytes.Equal(crcBytes, cache[len(cache)-4:]) {
				panic("invalid stream")
			}

			// LOG
			_DebugPrint(fmt.Sprintf("BngConn(%s): total bytes readed: %d", o._innerhid, len(dataWithoutCRC)))

			// Die Daten werden weiterverabeitet
			err = processCompleteData(o, dataWithoutCRC)
			if err != nil {
				_DebugPrint(fmt.Sprintf("BngConn(%s): Error processing final data: %v", o._innerhid, err))
				writePacketNACK(o) // Bei Fehler ein NACK senden
				readProcessErrorHandling(o, err)
				return
			}

			// Cache leeren, da die Verarbeitung abgeschlossen ist
			cache = cache[:0]

			// Es wird ein ACK an die Gegenseite gesendet
			err = writePacketACK(o)
			if err != nil {
				readProcessErrorHandling(o, err)
				_DebugPrint(fmt.Sprintf("BngConn(%s): Error sending pre-ACK for final message: %v", o._innerhid, err))
				return
			}
		case buffer[0] == byte(2):
			// Die Daten des Frames werden ohne das Erste Byte (Header) extrahiert und im cache zwischengespeichert
			cache = append(cache, buffer[1:sizeN]...)

			// LOG
			_DebugPrint(fmt.Sprintf("BngConn(%s): %d bytes frame recived", o._innerhid, len(buffer[:sizeN])))

			// Es wird versucht ein ACK an die Gegenseite zurückzusenden
			err := writePacketACK(o)
			if err != nil {
				readProcessErrorHandling(o, err)
				_DebugPrint(fmt.Sprintf("BngConn(%s): Error sending pre-ACK for final message: %v", o._innerhid, err))
				return
			}

			// Es wird auf den nächsten Datensatz gewartet
			continue
		case sizeN == 1:
			// Es wird geprüft um was es sich genau handelt
			switch buffer[0] {
			case 0: // ACK
				_DebugPrint(fmt.Sprintf("BngConn(%s): Received ACK", o._innerhid))
				o.ackHandle.EnterACK()
				continue
			case 1: // NACK
				_DebugPrint(fmt.Sprintf("BngConn(%s): Received NACK", o._innerhid))
				o.writerMutex.Lock()
				o.writerMutex.Unlock()
				continue
			}

			// Es wird auf den Nächsten Datensatz gewartet
			continue
		default:
		}
	}
}
