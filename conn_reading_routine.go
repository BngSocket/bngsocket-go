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
	totalRecived := 0
	_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was started", o._innerhid))

	for runningBackgroundServingLoop(o) {
		// Die Daten aus dem Paket werden eingelesen
		buffer := make([]byte, 41)
		sizeN, rerr := o.conn.Read(buffer)
		if rerr != nil {
			_DebugPrint(fmt.Sprintf("BngConn(%s): Error reading socket: %v", o._innerhid, rerr))
			readProcessErrorHandling(o, rerr)
			return
		}

		if sizeN < 1 {
			_DebugPrint(fmt.Sprintf("BngConn(%s): Invalid data stream, size < 1", o._innerhid))
			readProcessErrorHandling(o, fmt.Errorf("invalid data stream"))
			return
		}
		totalRecived += len(buffer[:sizeN])

		switch {
		case buffer[0] == byte(3): // Datenabschluss-Paket
			_DebugPrint(fmt.Sprintf("BngConn(%s): Flush received", o._innerhid))
			if len(cache) < 5 {
				_DebugPrint(fmt.Sprintf("BngConn(%s): Cache too small for CRC processing", o._innerhid))
				readProcessErrorHandling(o, fmt.Errorf("cache too small for CRC processing"))
				return
			}

			dataWithoutCRC := cache[:len(cache)-4]
			calculatedCRC := crc32.ChecksumIEEE(dataWithoutCRC)

			crcBytes := make([]byte, 4)
			binary.BigEndian.PutUint32(crcBytes, calculatedCRC)

			if !bytes.Equal(crcBytes, cache[len(cache)-4:]) {
				_DebugPrint(fmt.Sprintf("BngConn(%s): CRC mismatch", o._innerhid))
				panic("invalid stream")
			}

			_DebugPrint(fmt.Sprintf("BngConn(%s): Total bytes read: %d", o._innerhid, len(dataWithoutCRC)))
			err := processCompleteData(o, dataWithoutCRC)
			if err != nil {
				_DebugPrint(fmt.Sprintf("BngConn(%s): Error processing final data: %v", o._innerhid, err))
				writePacketNACK(o)
				readProcessErrorHandling(o, err)
				return
			}

			fmt.Println("TOTAL RECIVED", totalRecived)
			cache = cache[:0]
			totalRecived = 0
			err = writePacketACK(o)
			if err != nil {
				_DebugPrint(fmt.Sprintf("BngConn(%s): Error sending ACK: %v", o._innerhid, err))
				readProcessErrorHandling(o, err)
				return
			}
		case buffer[0] == byte(2): // Datenpaket
			cache = append(cache, buffer[1:sizeN]...)
			_DebugPrint(fmt.Sprintf("BngConn(%s): %d bytes frame received", o._innerhid, sizeN))

			err := writePacketACK(o)
			if err != nil {
				_DebugPrint(fmt.Sprintf("BngConn(%s): Error sending ACK: %v", o._innerhid, err))
				readProcessErrorHandling(o, err)
				return
			}
		case sizeN == 1: // Kontrollpaket (ACK/NACK)
			switch buffer[0] {
			case 0:
				_DebugPrint(fmt.Sprintf("BngConn(%s): Received ACK", o._innerhid))
				o.ackHandle.EnterACK()
			case 1:
				_DebugPrint(fmt.Sprintf("BngConn(%s): Received NACK", o._innerhid))
				o.writerMutex.Lock()
				o.writerMutex.Unlock()
			default:
				_DebugPrint(fmt.Sprintf("BngConn(%s): Unknown control packet: %d", o._innerhid, buffer[0]))
			}
		default:
			_DebugPrint(fmt.Sprintf("BngConn(%s): Unknown packet type received", o._innerhid))
			continue
		}
	}
}
