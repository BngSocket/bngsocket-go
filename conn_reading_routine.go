package bngsocket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

func handleMessage(reader *bufio.Reader, cache *bytes.Buffer, o *BngConn) error {
	// Lesen der Datenlänge (Big-Endian)
	var dataLength uint32
	if err := binary.Read(reader, binary.BigEndian, &dataLength); err != nil {
		return fmt.Errorf("%s: %w: %v", o._innerhid, ErrMessageLength, err)
	}

	// Lesen der Nachrichtendaten
	data := make([]byte, dataLength)
	if _, err := io.ReadFull(reader, data); err != nil {
		return fmt.Errorf("%s: %w: %v", o._innerhid, ErrMessageRead, err)
	}

	// Speichern der Daten im Cache
	cache.Write(data)

	// Berechne die Checksumme
	checksum := crc32.ChecksumIEEE(data)

	// Debug-Ausgabe: Checksumme und Länge der Nachricht
	_DebugPrint(fmt.Sprintf("BngConn(%s): MSG received, length=%d, checksum=%08x", o._innerhid, len(data), checksum))

	return nil
}

func handleEndTransfer(o *BngConn, cache *bytes.Buffer) error {
	// Berechne die Checksumme der Daten im Cache
	data := cache.Bytes()
	checksum := crc32.ChecksumIEEE(data)

	// Debug-Ausgabe: Länge und Checksumme der Daten
	_DebugPrint(fmt.Sprintf("BngConn(%s): ET received: Processing data (length=%d, checksum=%08x)", o._innerhid, len(data), checksum))

	// Starte die Verarbeitung in einer Goroutine
	o.backgroundProcesses.Add(1) // Informiere Wartungsgruppe
	go func(data []byte) {
		defer o.backgroundProcesses.Done() // Abschluss melden
		o._ProcessReadedData(data)         // Interne Verarbeitung
	}(data)

	// Cache leeren
	cache.Reset()

	// Erfolgreich verarbeitet
	return nil
}

func handleACK(reader *bufio.Reader, o *BngConn) error {
	ack := make([]byte, 2) // Das erste 'A' wurde schon gelesen
	if _, err := io.ReadFull(reader, ack); err != nil {
		return fmt.Errorf("%s: %w: %v", o._innerhid, ErrACKReadFailure, err)
	}

	// Prüfen, ob die Nachricht korrekt ist
	if string(ack) != "CK" {
		return fmt.Errorf("%s: %w", o._innerhid, ErrInvalidACK)
	}

	// Rufe die Methode `EnterACK` auf, um ACK zu bestätigen
	if err := o.ackHandle.EnterACK(); err != nil {
		return fmt.Errorf("%s: failed to process ACK: %v", o._innerhid, err)
	}

	_DebugPrint(fmt.Sprintf("BngConn(%s): ACK successfully processed", o._innerhid))
	return nil
}

func constantReading(o *BngConn) {
	defer func() {
		_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was stopped", o._innerhid))
		o.backgroundProcesses.Done()
	}()

	_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was started", o._innerhid))

	reader := bufio.NewReader(o.conn)
	var cache bytes.Buffer // Cache für MSG-Daten

	for runningBackgroundServingLoop(o) {
		// Lesen des Typ-Bytes
		msgType, err := reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				_DebugPrint(fmt.Sprintf("BngConn(%s): %v", o._innerhid, ErrConnectionClosed))
				break
			}
			_DebugPrint(fmt.Sprintf("BngConn(%s): %v: %v", o._innerhid, ErrReadMessageType, err))
			return
		}

		switch msgType {
		case 'M': // MSG: Ein Teil des Datensatzes
			_DebugPrint(fmt.Sprintf("BngConn(%s): MSG received", o._innerhid))
			if err := handleMessage(reader, &cache, o); err != nil {
				_DebugPrint(fmt.Sprintf("BngConn(%s): %v: %v", o._innerhid, ErrProcessingMessage, err))
				return
			}
			// Sende ACK zurück
			if err := writePacketACK(o); err != nil {
				_DebugPrint(fmt.Sprintf("BngConn(%s): failed to send ACK after MSG: %v", o._innerhid, err))
				return
			}
		case 'E': // ET: Ende des Datensatzes
			_DebugPrint(fmt.Sprintf("BngConn(%s): ET received", o._innerhid))
			// Jetzt den kompletten Datensatz aus dem Cache verarbeiten
			if err := handleEndTransfer(o, &cache); err != nil {
				_DebugPrint(fmt.Sprintf("BngConn(%s): %v: %v", o._innerhid, ErrProcessingEndTransfer, err))
				return
			}
			// Sende ACK zurück
			if err := writePacketACK(o); err != nil {
				_DebugPrint(fmt.Sprintf("BngConn(%s): failed to send ACK after ET: %v", o._innerhid, err))
				return
			}
		case 'A': // ACK: Eingehende Bestätigung
			_DebugPrint(fmt.Sprintf("BngConn(%s): ACK received", o._innerhid))
			if err := handleACK(reader, o); err != nil {
				_DebugPrint(fmt.Sprintf("BngConn(%s): %v: %v", o._innerhid, ErrProcessingACK, err))
				return
			}
		default:
			_DebugPrint(fmt.Sprintf("BngConn(%s): %v: %c", o._innerhid, ErrUnknownMessageType, msgType))
		}
	}
}
