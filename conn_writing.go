package bngsocket

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

func writeBytesAndWaitOfACK(o *BngConn, data []byte) error {
	if len(data) > 4096 {
		return fmt.Errorf("data to big")
	}

	totalSend := 0
	for totalSend < len(data) {
		n, err := o.conn.Write(data)
		if err != nil {
			return err
		}
		totalSend += n
		if err := o.ackHandle.WaitOfACK(); err != nil {
			return err
		}
	}
	return nil
}

// writeBytesIntoSocketConn schreibt ein Byte-Array in den Schreibkanal des angegebenen Sockets.
// Es gibt einen Fehler zurück, wenn der Socket oder der Schreibkanal nicht verfügbar ist.
func writeBytesIntoSocketConn(o *BngConn, nData []byte) error {
	// Der Writing Mutex wird verwendet
	o.writerMutex.Lock()
	defer o.writerMutex.Unlock()

	const frameSize = 40 // Maximale Größe eines Frames

	// Berechne die CRC32-Prüfsumme der gesamten Daten
	crc := crc32.ChecksumIEEE(nData)
	crcBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBytes, crc)

	// Daten mit CRC zusammenfügen
	data := append(nData, crcBytes...)

	// Das Steuerbyte wird übertragen, dieses Signalisiert dass eine neue Übertragung gestartet werden soll
	_DebugPrint(fmt.Sprintf("BngConn(%s): write control byte", o._innerhid))
	if err := writeBytesAndWaitOfACK(o, []byte{0}); err != nil {
		return err
	}

	// Die Größe der Daten werden übertragen
	sizeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(sizeBytes, uint64(len(data)))

	// Die Gesamtgröße des Datensatzes wird übertragen
	_DebugPrint(fmt.Sprintf("BngConn(%s): write data size", o._innerhid))
	if err := writeBytesAndWaitOfACK(o, sizeBytes); err != nil {
		return err
	}

	totalBytes := len(data) // Gesamtgröße der zu übertragenden Daten
	bytesSent := 0          // Anzahl der bereits gesendeten Bytes

	// Übertragung der Daten in Paketen
	for bytesSent < totalBytes {
		// Berechne die Größe des aktuellen Chunks
		chunkSize := frameSize
		if totalBytes-bytesSent < chunkSize {
			chunkSize = totalBytes - bytesSent
		}

		// Paket schreiben
		err := writeBytesAndWaitOfACK(o, (data[bytesSent : bytesSent+chunkSize]))
		if err != nil {
			writeProcessErrorHandling(o, fmt.Errorf("error writing data to socket: %w", err))
			return err
		}
	}

	_DebugPrint(fmt.Sprintf("BngConn(%s): write flush byte: byte(3)", o._innerhid))

	// Flush-Byte senden
	err := writeBytesAndWaitOfACK(o, []byte{'\n'})
	if err != nil {
		writeProcessErrorHandling(o, fmt.Errorf("flush byte write error: %w", err))
		return err
	}

	// LOG
	_DebugPrint(fmt.Sprintf("BngConn(%s): total bytes written: %d", o._innerhid, bytesSent))
	return nil
}
