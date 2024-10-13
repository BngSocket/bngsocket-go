package bngsocket

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// Gibt die SitzungsId zurück
func (o *BngConnChannel) GetSessionId() string {
	return o.sesisonId
}

// Read implementiert die Read-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) Read(b []byte) (n int, err error) {
	// Überprüfe zu Beginn, ob der Kanal oder die Verbindung geschlossen ist oder ein Fehler vorliegt
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("BngConnChannel->Read: %w", runningErr)
	}

	// Beginne einen Lesevorgang
	m.openReaders.Add(1)
	defer m.openReaders.Sub(1)

	// Stelle sicher, dass nur ein Lesevorgang aktiv ist
	if m.openReaders.Get() > 1 {
		return 0, ErrConcurrentReadingNotAllowed
	}

	// Prüfe, ob Daten im Cache sind
	if cachedData := m.currentReadingCache.Get(); len(cachedData) > 0 {
		// Die Daten aus dem Cache werden kopiert
		if len(b) < len(cachedData) {
			n = copy(b, cachedData[:len(b)])
			m.currentReadingCache.Set(cachedData[len(b):]) // Speichere restliche Daten im Cache
		} else {
			n = copy(b, cachedData)
			m.currentReadingCache.Set(nil) // Leere den Cache
		}

		// Überprüfe erneut den Kanalzustand
		if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
			if channClosed || connClosed {
				return n, io.EOF
			}
			return n, fmt.Errorf("BngConnChannel->Read: %w", runningErr)
		}

		return n, nil
	}

	// Wenn keine Daten im Cache sind, lese neue Daten
	bytes, pid, err := m.bytesDataInCache.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("BngConnChannel->Read: %w", err)
	}

	// Überprüfe erneut den Kanalzustand nach dem Lesen
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("BngConnChannel->Read: %w", runningErr)
	}

	// Kopiere die gelesenen Daten in den Puffer b
	if len(b) < len(bytes) {
		n = copy(b, bytes[:len(b)])
		m.currentReadingCache.Set(bytes[len(b):]) // Speichere restliche Bytes im Cache
	} else {
		n = copy(b, bytes)
		m.currentReadingCache.Set(nil)
	}

	// Überprüfe erneut den Kanalzustand
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return n, io.EOF
		}
		return n, fmt.Errorf("BngConnChannel->Read: %w", runningErr)
	}

	// Sende ein ACK zurück
	if err := channelWriteACK(m.socket, pid, m.sesisonId); err != nil {
		if errors.Is(err, io.EOF) {
			return n, io.EOF
		}
		return n, fmt.Errorf("BngConnChannel->Read: %w", err)
	}

	// Überprüfe erneut den Kanalzustand
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return n, io.EOF
		}
		return n, fmt.Errorf("BngConnChannel->Read: %w", runningErr)
	}

	return n, nil
}

// Write implementiert die Write-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) Write(b []byte) (n int, err error) {
	// Es wird geprüft, ob der Channel geschlossen ist oder ein Fehler vorliegt
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("BngConnChannel->Write: %w", runningErr)
	}

	// Es wird eine Lese Funktion hinzugefügt
	m.openWriters.Add(1)
	defer m.openWriters.Sub(1)

	// Es wird geprüft, wie viele Lesevorgänge derzeit aktiv sind
	if m.openWriters.Get() > 1 {
		return 0, ErrConcurrentWritingNotAllowed // Es wird ein Fehler ausgelöst
	}

	// Es wird versucht, die Daten in den Channel zu schreiben
	packageId, writtenSize, err := channelDataTransport(m.socket, b, m.sesisonId) // sessionId korrigiert
	if err != nil {
		if errors.Is(err, io.EOF) {
			return 0, err
		}
		return 0, fmt.Errorf("BngConnChannel->Write: %w", err) // %w für Error-Wrapping
	}

	// Es wird erneut geprüft, ob der Channel geschlossen wurde oder ein Fehler vorliegt
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("BngConnChannel->Write: %w", runningErr)
	}

	// Der Status des Channels wird auf "WaitOfACK" gesetzt
	m.waitOfPackageACK.Set(true)

	// Es wird auf die Bestätigung durch die Gegenseite gewartet
	ackPackageId, ok := m.ackChan.Read()
	if !ok {
		// Es wird geprüft, ob der Socket geschlossen wurde
		if m.isClosed.Get() {
			return 0, io.EOF
		}

		// Es wird geprüft, ob der ACK-Channel geschlossen wurde
		if !m.ackChan.IsOpen() {
			return 0, fmt.Errorf("BngConnChannel->Write: ACK waiting channel was closed")
		}

		// Ein unerwarteter Fehlerzustand beim Warten auf ACK
		return 0, fmt.Errorf("BngConnChannel->Write: unexpected error while waiting for ACK")
	}

	// Es wird erneut geprüft, ob der Channel geschlossen wurde oder ein Fehler vorliegt
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("BngConnChannel->Write: %w", runningErr)
	}

	// Die Paket-ID wird mit der gesendeten ID überprüft
	if ackPackageId.pid != packageId {
		// Der Fehler wird erzeugt
		ferr := fmt.Errorf("BngConnChannel->Write: ACK channel error, invalid package ID. Expected: %d, got: %d", packageId, ackPackageId.pid)

		// Es handelt sich um ein Protokollfehler, die Hauptverbindung muss aus sicherheitsgründen geschlossen werden.
		// Mit dem Schließen der Hauptverbindung ist der Channel nicht mehr verwendetbar.
		m.socket._ConsensusProtocolTermination(ferr)

		// Der Channel wird ohne Signal geschlossen
		m.processClose(false)

		// Der Fehler wird zurückgegeben
		return 0, ferr
	}

	// Die Daten wurden erfolgreich übertragen
	return writtenSize, nil
}

// Close implementiert die Close-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) Close() error {
	// Es wird geprüft ob das Aktuelle Objket bereits geschlossen wurde
	if m.isClosed.Get() {
		return fmt.Errorf("is always closed")
	}

	// Der Eigentlich Close Process wird durchgeführt
	if err := m.processClose(true); err != nil {
		return err
	}

	// Es ist kein Fehler aufgetreten
	return nil
}

// LocalAddr implementiert die LocalAddr-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) LocalAddr() net.Addr {
	return m.socket.LocalAddr()
}

// RemoteAddr implementiert die RemoteAddr-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) RemoteAddr() net.Addr {
	return m.socket.RemoteAddr()
}

// SetDeadline implementiert die SetDeadline-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline implementiert die SetReadDeadline-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline implementiert die SetWriteDeadline-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) SetWriteDeadline(t time.Time) error {
	return nil
}

// Führt eine Reihe an Standardaufgaben durch wenn das Objekt geschlossen werden soll
func (m *BngConnChannel) processClose(sendSignal bool) error {
	// Der Objekt Mutex wird verwendet
	m.mu.Lock()
	defer m.mu.Unlock()

	// Das Objekt wird (gleich MAX_OPERATIONS = 1) geschlossen
	if m.isClosed.Set(true) != 1 {
		return io.EOF
	}

	// Wird am ende ausgeführt um sicherzustellen das der Channel vernichtet wird
	defer func() {
		// Der BngConn wird mitgeteilt dass der Channl geschlosen wurde
		if err := m.socket._UnregisterChannelSession(m.sesisonId); err != nil {
			panic("BngConnChannel->Close: " + err.Error())
		}
	}()

	// Die ACK Chan wird geschlosen
	m.ackChan.Close()
	m.bytesDataInCache.Close()

	// Es wird ein Close Paket an die Gegenseite gesendet
	if sendSignal {
		if err := channelWriteCloseSignal(m.socket, m.sesisonId); err != nil {
			return fmt.Errorf("BngConnChannel->Close: " + err.Error())
		}
	}

	// Debug
	DebugPrint("Channel is closed:", m.sesisonId)

	// Es ist Kein Fehler aufgetreten
	return nil
}

// Nimmt eingetroffene Datensätze entgegen
func (m *BngConnChannel) enterIncommingData(data []byte, packageId uint64) error {
	// Es wird geprüft ob der Channel, die Conn geschlossen ist oder ein Runnig fehler vorhanden ist
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return io.EOF
		}
		return fmt.Errorf("BngConnChannel->enterIncommingData: %w", runningErr)
	}

	// Die Daten werden in den Cache geschrieben
	if werr := m.bytesDataInCache.Write(data, packageId); werr != nil {
		if errors.Is(werr, io.EOF) {
			return io.EOF
		}
		return fmt.Errorf("BngConnChannel->enterIncommingData: %w", werr)
	}

	// Es wird geprüft ob der Channel, die Conn geschlossen ist oder ein Runnig fehler vorhanden ist
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return io.EOF
		}
		return fmt.Errorf("BngConnChannel->enterIncommingData: %w", runningErr)
	}

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um den Status einer Paket übertragung zu Signalisieren
func (m *BngConnChannel) enterChannelTransportStateResponseSate(packageId uint64, state uint8) error {
	// Es wird geprüft ob der Channel, die Conn geschlossen ist oder ein Runnig fehler vorhanden ist
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return io.EOF
		}
		return fmt.Errorf("BngConnChannel->enterChannelTransportStateResponseSate: %w", runningErr)
	}

	// Es wird geprüft ob auf ein ACK Paket gewartet wird
	if !m.waitOfPackageACK.Get() {
		return fmt.Errorf("BngConnChannel->enterChannelTransportStateResponseSate: no waiting for ack")
	}

	// Es wird geprüft ob der Channel zuletzt das Paket mit der ID abgesendet hat
	if ok := m.ackChan.Enter(&AckItem{pid: packageId, state: state}); !ok {
		return io.EOF
	}

	// Es wird geprüft ob der Channel, die Conn geschlossen ist oder ein Runnig fehler vorhanden ist
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return io.EOF
		}
		return fmt.Errorf("BngConnChannel->enterChannelTransportStateResponseSate: %w", runningErr)
	}

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um ein Siganl von der Gegenseite auszuwerten
func (m *BngConnChannel) enterSignal(signalId uint64) error {
	// Es wird geprüft ob der Channel, die Conn geschlossen ist oder ein Runnig fehler vorhanden ist
	if channClosed, connClosed, runningErr := _IsClosedOrHasRunningErrorOnChannel(m); channClosed || connClosed || runningErr != nil {
		if channClosed || connClosed {
			return io.EOF
		}
		return fmt.Errorf("BngConnChannel->enterSignal: %w", runningErr)
	}

	// Es wird geprüft ob es sich um ein bekanntes Signal handelt
	switch signalId {
	case 0:
		// Der Channel wird geschlossen
		if err := m.processClose(false); err != nil {
			return fmt.Errorf("BngConnChannel->enterSignal: " + err.Error())
		}
	default:
		fmt.Println("UNKOWN SIG")
		return fmt.Errorf("BngConnChannel->enterSignal: unkown signal")
	}

	// Es ist kein Fehler aufgetreten
	return nil
}
