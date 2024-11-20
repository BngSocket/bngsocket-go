package bngsocket

import (
	"encoding/binary"
	"fmt"
)

// writeBytesIntoSocketConn schreibt ein Byte-Array in den Schreibkanal des angegebenen Sockets.
// Es gibt einen Fehler zurück, wenn der Socket oder der Schreibkanal nicht verfügbar ist.
func writeBytesIntoSocketConn(o *BngConn, data []byte) error {
	const chunkSize = 1024 // Maximale Größe eines Chunks in Bytes

	// Gesamtlänge der Daten
	totalLength := len(data)
	_DebugPrint(fmt.Sprintf("BngConn(%s): Sending %d bytes in %d-byte chunks", o._innerhid, totalLength, chunkSize))

	// Aufteilen und Senden der Daten
	for start := 0; start < totalLength; start += chunkSize {
		end := start + chunkSize
		if end > totalLength {
			end = totalLength // Der letzte Chunk hat möglicherweise weniger als 1024 Bytes
		}

		chunk := data[start:end]

		// Kritischer Abschnitt: Sperre den Mutex
		o.connMutex.Lock()
		defer o.connMutex.Unlock()

		// Schreibe den Typ 'M' (Message)
		if err := o.writer.WriteByte('M'); err != nil {
			errmsg := fmt.Errorf("%w: %v", ErrWriteMessageType, err)
			writeProcessErrorHandling(o, errmsg)
			return errmsg
		}

		// Schreibe die Länge des Chunks (Big-Endian)
		chunkLength := uint32(len(chunk))
		if err := binary.Write(o.writer, binary.BigEndian, chunkLength); err != nil {
			errmsg := fmt.Errorf("%w: %v", ErrWriteChunkLength, err)
			writeProcessErrorHandling(o, errmsg)
			return errmsg
		}

		// Schreibe den Chunk selbst
		bytesToWrite := len(chunk)
		for bytesWritten := 0; bytesWritten < bytesToWrite; {
			n, err := o.writer.Write(chunk[bytesWritten:])
			if err != nil {
				errmsg := fmt.Errorf("%w: %v", ErrWriteChunk, err)
				writeProcessErrorHandling(o, errmsg)
				return errmsg
			}

			bytesWritten += n
			if n == 0 {
				errmsg := fmt.Errorf("%w: no further bytes written, connection may be broken", ErrWriteChunk)
				writeProcessErrorHandling(o, errmsg)
				return errmsg
			}
		}

		// Flush die Daten
		if err := o.writer.Flush(); err != nil {
			errmsg := fmt.Errorf("%w: %v", ErrFlushWriter, err)
			writeProcessErrorHandling(o, errmsg)
			return errmsg
		}

		_DebugPrint(fmt.Sprintf("BngConn(%s): Chunk [%d:%d] sent", o._innerhid, start, end))

		// Mutex freigeben, bevor auf ACK gewartet wird
		o.connMutex.Unlock()
		defer o.connMutex.Lock()

		// Warte auf ACK
		if err := o.ackHandle.WaitOfACK(); err != nil {
			errmsg := fmt.Errorf("%w for chunk [%d:%d]: %v", ErrWaitForACK, start, end, err)
			writeProcessErrorHandling(o, errmsg)
			return errmsg
		}
	}

	// Senden von EndTransfer (ET)
	o.connMutex.Lock()
	defer o.connMutex.Unlock()

	if err := o.writer.WriteByte('E'); err != nil {
		errmsg := fmt.Errorf("%w: %v", ErrWriteEndTransfer, err)
		writeProcessErrorHandling(o, errmsg)
		return errmsg
	}
	if err := o.writer.Flush(); err != nil {
		errmsg := fmt.Errorf("%w after ET: %v", ErrFlushWriter, err)
		writeProcessErrorHandling(o, errmsg)
		return errmsg
	}

	_DebugPrint(fmt.Sprintf("BngConn(%s): ET sent", o._innerhid))

	// Warten auf ACK für ET
	if err := o.ackHandle.WaitOfACK(); err != nil {
		errmsg := fmt.Errorf("%w after ET: %v", ErrWaitForACK, err)
		writeProcessErrorHandling(o, errmsg)
		return errmsg
	}

	return nil
}
