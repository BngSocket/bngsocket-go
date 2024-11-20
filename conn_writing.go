package bngsocket

import (
	"bufio"
	"encoding/binary"
	"fmt"
)

// writeBytesIntoSocketConn schreibt ein Byte-Array in den Schreibkanal des angegebenen Sockets.
// Es gibt einen Fehler zurück, wenn der Socket oder der Schreibkanal nicht verfügbar ist.
func writeBytesIntoSocketConn(o *BngConn, data []byte) error {
	const chunkSize = 1024 // Maximale Größe eines Chunks in Bytes
	writer := bufio.NewWriter(o.conn)

	// Gesamtlänge der Daten
	totalLength := len(data)
	_DebugPrint(fmt.Sprintf("BngConn(%s): Sending %d bytes in %d-byte chunks", o._innerhid, totalLength, chunkSize))

	// Aufteilen und Senden der Daten
	for start := 0; start < totalLength; start += chunkSize {
		end := start + chunkSize
		if end > totalLength {
			end = totalLength // Der letzte Chunk hat möglicherweise weniger als 64 Bytes
		}

		chunk := data[start:end]

		// Schreibe den Typ 'M' (Message)
		if err := writer.WriteByte('M'); err != nil {
			return fmt.Errorf("%w: %v", ErrWriteMessageType, err)
		}

		// Schreibe die Länge des Chunks (Big-Endian)
		chunkLength := uint32(len(chunk))
		if err := binary.Write(writer, binary.BigEndian, chunkLength); err != nil {
			return fmt.Errorf("%w: %v", ErrWriteChunkLength, err)
		}

		// Schreibe den Chunk selbst
		bytesToWrite := len(chunk)
		for bytesWritten := 0; bytesWritten < bytesToWrite; {
			n, err := writer.Write(chunk[bytesWritten:])
			if err != nil {
				return fmt.Errorf("%w: %v", ErrWriteChunk, err)
			}

			bytesWritten += n
			if n == 0 {
				return fmt.Errorf("%w: no further bytes written, connection may be broken", ErrWriteChunk)
			}
		}

		// Flush die Daten
		if err := writer.Flush(); err != nil {
			return fmt.Errorf("%w: %v", ErrFlushWriter, err)
		}

		_DebugPrint(fmt.Sprintf("BngConn(%s): Chunk [%d:%d] sent", o._innerhid, start, end))

		// Warte auf ACK
		if err := o.ackHandle.WaitOfACK(); err != nil {
			return fmt.Errorf("%w for chunk [%d:%d]: %v", ErrWaitForACK, start, end, err)
		}
	}

	// Senden von EndTransfer (ET)
	if err := writer.WriteByte('E'); err != nil {
		return fmt.Errorf("%w: %v", ErrWriteEndTransfer, err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("%w after ET: %v", ErrFlushWriter, err)
	}
	_DebugPrint(fmt.Sprintf("BngConn(%s): ET sent", o._innerhid))

	// Warten auf ACK für ET
	if err := o.ackHandle.WaitOfACK(); err != nil {
		return fmt.Errorf("%w after ET: %v", ErrWaitForACK, err)
	}

	return nil
}
