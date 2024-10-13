package bngsocket

import (
	"bytes"
	"io"
	"sync"
)

func newBngConnChannelByteCache() *ByteCache {
	bc := &ByteCache{
		dataItems: []*_DataItem{},
		currentID: 0,
		closed:    false,
	}
	bc.cond = sync.NewCond(&bc.mu)
	return bc
}

// Write schreibt Daten in den ByteCache.
func (bc *ByteCache) Write(data []byte, id uint64) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Es wird geprüft ob der ByteChannel geschlossen wurde
	if bc.closed {
		return io.EOF
	}

	// Erstellen eines neuen Datensatzes mit einer einzigartigen ID und Speichern der Daten.
	item := &_DataItem{
		id:        id,
		data:      bytes.NewBuffer(data),
		totalSize: len(data),
		bytesRead: 0,
	}

	// Datensatz dem Cache hinzufügen.
	bc.dataItems = append(bc.dataItems, item)

	// Signalisiert, dass jetzt Daten verfügbar sind.
	bc.cond.Signal()

	// Der Vorgagng war erfolgreich
	return nil
}

// ReadAll liest den gesamten aktuellen Datensatz und gibt die ID zurück.
func (bc *ByteCache) Read() ([]byte, uint64, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Es wird geprüft ob der ByteChannel geschlossen wurde
	if bc.closed {
		return nil, 0, io.EOF
	}

	// Warten, bis Daten im Cache vorhanden sind.
	for len(bc.dataItems) == 0 {
		bc.cond.Wait() // Blockiert, bis Daten vorhanden sind.
	}

	// Den gesamten aktuellen Datensatz lesen.
	currentItem := bc.dataItems[0]
	data := currentItem.data.Bytes() // Liest alle verbleibenden Bytes.
	id := currentItem.id

	// Datensatz aus der Liste entfernen.
	bc.dataItems = bc.dataItems[1:]

	// Rückgabe des gesamten Datensatzes und der ID.
	return data, id, nil // Nach vollständigem Lesen wird io.EOF zurückgegeben.
}

// Gibt an ob das Objekt geschlossen wurde
func (bc *ByteCache) IsClosed() bool {
	bc.mu.Lock()
	v := bc.closed
	bc.mu.Unlock()
	return v
}

// Wird Verwendet um den Cache zu schließen
func (bc *ByteCache) Close() {
	bc.mu.Lock()
	bc.closed = true
	bc.mu.Unlock()
}
