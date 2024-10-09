package bngsocket

import (
	"fmt"
	"net"
	"time"
)

// Gibt die SitzungsId zurück
func (o *BngConnChannel) GetSessionId() string {
	return o.sesisonId
}

// Read implementiert die Read-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) Read(b []byte) (n int, err error) {
	// Es wird eine Lese Funktion hinzugefügt
	m.openReaders.Add(1)

	// Es wird geprüft, wie viele Lesevorgänge derzeit aktiv sind
	if m.openReaders.Get() > 1 {
		m.openReaders.Sub(1) // Nicht vergessen, wieder zu subtrahieren
		return 0, fmt.Errorf("simultanes reading not allowed")
	}

	// Nachdem Lesen wird 1 Minus gerechnet
	defer m.openReaders.Sub(1)

	// Es wird geprüft, ob derzeit ein Datensatz zum Auslesen vorhanden ist
	readedCache := m.currentReadingCache.Get()
	if readedCache == nil {
		// Die aktuellen Daten werden alle ausgelesen
		bytes, pid, err := m.bytesDataInCache.Read()
		if err != nil {
			return 0, fmt.Errorf("BngConnChannel->Read: %s", err.Error())
		}

		// Es wird geprüft ob die Verbindung geschlossen wurde
		if m.isClosed.Get() {

		}

		// Die Daten werden im aktuellen Cache zwischengespeichert, aber nur bis zur Länge des Buffers b
		if len(b) < len(bytes) {
			m.currentReadingCache.Set(bytes[len(b):]) // Speichert nur die Bytes, welche nicht in b geschrieben werden
			n = copy(b, bytes[:len(b)])               // Kopiere die Bytes bis zur Länge von b
		} else {
			n = copy(b, bytes)
		}

		// Es wird geprüft ob die Verbindung geschlossen wurde
		if m.isClosed.Get() {

		}

		// Es wird ein ACK zurückgesendet
		if err := channelWriteACK(m.socket, pid, m.sesisonId); err != nil {
			return 0, fmt.Errorf("BngConnChannel->Read: %s", err.Error())
		}

		// Die Gesamtlänge wird zurückgegeben
		return n, nil // Anzahl der kopierten Bytes zurückgeben
	}

	// Falls bereits Daten im Cache sind, diese zurückgeben
	if len(b) < len(readedCache) {
		m.currentReadingCache.Set(readedCache[len(b):]) // Speichert nur die Bytes, welche nicht in b geschrieben werden
		n = copy(b, readedCache[:len(b)])               // Kopiere die Bytes bis zur Länge von b
	} else {
		m.currentReadingCache.Set(nil)
		n = copy(b, readedCache)
	}

	// Es wird geprüft ob die Verbindung geschlossen wurde
	if m.isClosed.Get() {

	}

	// Das Ergebniss wird zurückgegeben
	return n, nil
}

// Write implementiert die Write-Methode des net.Conn-Interfaces.
func (m *BngConnChannel) Write(b []byte) (n int, err error) {
	// Es wird geprüft ob die Verbindung geschlossen wurde
	if m.isClosed.Get() {

	}

	// Es wird versucht die Daten in den Channl zu schreiben
	packageId, writedSize, err := channelDataTransport(m.socket, b, m.sesisonId)
	if err != nil {
		return 0, fmt.Errorf("BngConnChannel->Write: " + err.Error())
	}

	// Der Stauts des Channels wird auf "WaitOfACK" gesetzt
	m.waitOfPackageACK.Set(true)

	// Es wird auf die Bestätigung durch die gegenseite gewartet
	ackPackageId, ok := m.ackChan.Read()
	if !ok {
		return 0, fmt.Errorf("BngConnChannel->Write: ack memory error")
	}

	// Die Paket ID wird mit der ID überprüft, welche zuletzt gesendet wurde
	if ackPackageId.pid != packageId {
		return 0, fmt.Errorf("BngConnChannel->Write: ack channel error, invalid package id")
	}

	// Die Daten wurden erfolgreich übertragen
	return writedSize, nil
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
	// Das Objekt wird (gleich MAX_OPERATIONS = 1) geschlossen
	if m.isClosed.Set(true) != 1 {
		return fmt.Errorf("is always closed")
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

	// Es wird ein Close Paket an die Gegenseite gesendet
	if sendSignal {
		if err := channelWriteCloseSignal(m.socket, m.sesisonId); err != nil {
			return fmt.Errorf("BngConnChannel->Close: " + err.Error())
		}
	}

	// Es ist Kein Fehler aufgetreten
	return nil
}

// Nimmt eingetroffene Datensätze entgegen
func (m *BngConnChannel) enterIncommingData(data []byte, packageId uint64) error {
	// Es wird geprüft ob der Channl noch geöffnet ist
	if m.isClosed.Get() {
		return fmt.Errorf("channel was closed")
	}

	// Es wird geprüft ob Datensätze Empfangen werden dürfen oder ob auf ein ACK für das Letzte Paket gewaret wird
	if m.waitOfPackageACK.Get() {
		// Der Vorgang wird abgebrochen, da der Channel auf ein ACK wartet
		return fmt.Errorf("wait of ack, data reciving in current moment, not allowed")
	}

	// Die Daten werden in den Cache geschrieben
	m.bytesDataInCache.Write(data, packageId)

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um den Status einer Paket übertragung zu Signalisieren
func (m *BngConnChannel) enterChannelTransportStateResponseSate(packageId uint64, state uint8) error {
	// Es wird geprüft ob auf ein ACK Paket gewartet wird
	if !m.waitOfPackageACK.Get() {
		return fmt.Errorf("BngConnChannel->enterChannelTransportStateResponseSate: no waiting for ack")
	}

	// Es wird geprüft ob der Channel zuletzt das Paket mit der ID abgesendet hat
	m.ackChan.Enter(&AckItem{pid: packageId, state: state})

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um ein Siganl von der Gegenseite auszuwerten
func (m *BngConnChannel) enterSignal(signalId uint64) error {
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
