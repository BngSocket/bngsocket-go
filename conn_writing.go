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

	const packetSize = 4095 // Maximale Größe eines Pakets

	// Berechne die CRC32-Prüfsumme der gesamten Daten
	crc := crc32.ChecksumIEEE(nData)
	crcBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBytes, crc)

	// Es wird ein neuer Datensatz mit der CRC32 Checksumme erzeugt
	data := make([]byte, 0)
	data = append(data, nData...)
	data = append(data, crcBytes...)

	totalBytes := len(data) // Gibt die Gesamtgröße der zu Übertragenden Bytes an
	bytesSent := 0          // Gibt an, wv Bytes bereits übertragen wurden

	// Übertragung der Daten in Paketen
	for bytesSent < totalBytes {
		// Berechne die Größe des nächsten Pakets
		remainingBytes := totalBytes - bytesSent
		chunkSize := packetSize
		if remainingBytes < packetSize {
			chunkSize = remainingBytes
		}

		// Extrahiere das nächste Paket
		packet := []byte{2}
		packet = append(packet, data[bytesSent:bytesSent+chunkSize]...)

		// Es wird versucht das Paket in den Socket zu schreiben
		n, err := o.conn.Write(packet)
		if err != nil {
			writeProcessErrorHandling(o, err)
			return err
		}
		if n-1 < 1 {
			panic("unkown error")
		}

		// LOG
		_DebugPrint(fmt.Sprintf("BngConn(%s): %d bytes frame writed", o._innerhid, n))

		// Es wird auf ein ACK gewartet
		if err := o.ackHandle.WaitOfACK(); err != nil {
			writeProcessErrorHandling(o, err)
			return err
		}

		// Bytes send wird um die Größe der Wirklch gesendeten Bytes erweitert
		bytesSent += n - 1
	}

	// LOG
	_DebugPrint(fmt.Sprintf("BngConn(%s): write flusch byte: byte(3)", o._innerhid))

	// Es wird bestätigt das es sich um das letze Paket handelt
	n, err := o.conn.Write([]byte{3})
	if err != nil {
		writeProcessErrorHandling(o, err)
		return err
	}
	if n != 1 {
		fmt.Println(n - 1)
		panic("unknown error")
	}

	// Es wird auf ein ACK gewartet
	if err := o.ackHandle.WaitOfACK(); err != nil {
		writeProcessErrorHandling(o, err)
		return err
	}

	// LOG
	_DebugPrint(fmt.Sprintf("BngConn(%s): total bytes writed: %d", o._innerhid, len(data)))

	return nil
}
