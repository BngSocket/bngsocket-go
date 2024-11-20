package bngsocket

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

// writeBytesIntoSocketConn schreibt ein Byte-Array in den Schreibkanal des angegebenen Sockets.
// Es gibt einen Fehler zurück, wenn der Socket oder der Schreibkanal nicht verfügbar ist.
func writeBytesIntoSocketConn(o *BngConn, nData []byte) error {
	// Der Writing Mutex wird verwendet
	o.writerMutex.Lock()
	defer o.writerMutex.Unlock()

	const packetSize = 40 // Maximale Größe eines Pakets

	// Berechne die CRC32-Prüfsumme der gesamten Daten
	crc := crc32.ChecksumIEEE(nData)
	crcBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBytes, crc)

	// Daten mit CRC zusammenfügen
	data := append(nData, crcBytes...)

	totalBytes := len(data) // Gesamtgröße der zu übertragenden Daten
	bytesSent := 0          // Anzahl der bereits gesendeten Bytes
	totalSend := 0

	_DebugPrint(fmt.Sprintf("BngConn(%s): write %d bytes", o._innerhid, len(nData)))

	// Übertragung der Daten in Paketen
	for bytesSent < totalBytes {
		remainingBytes := totalBytes - bytesSent
		chunkSize := packetSize
		if remainingBytes < packetSize {
			chunkSize = remainingBytes
		}

		// Paket erstellen
		packet := []byte{2}
		packet = append(packet, data[bytesSent:bytesSent+chunkSize]...)

		// Paket schreiben
		fmt.Println("WRITED_PACKAGE")
		n, err := o.conn.Write(packet)
		if err != nil {
			writeProcessErrorHandling(o, err)
			return fmt.Errorf("error writing data to socket: %w", err)
		}
		totalSend += n
		if n < 1 {
			return fmt.Errorf("unexpected write error: wrote fewer bytes than expected (%d)", n-1)
		}

		_DebugPrint(fmt.Sprintf("BngConn(%s): Successfully wrote frame of %d bytes (chunk size: %d, total sent: %d/%d)", o._innerhid, n, chunkSize, bytesSent+chunkSize, totalBytes))

		// Auf ACK warten
		if err := o.ackHandle.WaitOfACK(); err != nil {
			writeProcessErrorHandling(o, fmt.Errorf("error waiting for ACK: %w", err))
			return err
		}

		// Aktualisieren der gesendeten Bytes
		bytesSent += n - 1
	}

	fmt.Println("TOTAL SEND:", totalSend)

	_DebugPrint(fmt.Sprintf("BngConn(%s): write flush byte: byte(3)", o._innerhid))

	// Flush-Byte senden
	n, err := o.conn.Write([]byte{3})
	if err != nil {
		writeProcessErrorHandling(o, fmt.Errorf("flush byte write error: %w", err))
		return err
	}
	if n != 1 {
		return fmt.Errorf("flush byte write error: expected 1 byte, wrote %d", n)
	}

	// Auf abschließendes ACK warten
	if err := o.ackHandle.WaitOfACK(); err != nil {
		writeProcessErrorHandling(o, fmt.Errorf("error waiting for final ACK: %w", err))
		return err
	}

	_DebugPrint(fmt.Sprintf("BngConn(%s): total bytes written: %d", o._innerhid, len(nData)))

	return nil
}
