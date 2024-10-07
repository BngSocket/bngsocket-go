package bngsocket

import (
	"bytes"
	"sync"
)

func newBngConnChannelByteCache() *_BngConnChannelByteCache {
	bc := &_BngConnChannelByteCache{
		dataItems: []*_DataItem{},
		currentID: 0,
	}
	bc.cond = sync.NewCond(&bc.mu)
	return bc
}

// Write schreibt Daten in den ByteCache.
func (bc *_BngConnChannelByteCache) Write(data []byte, id uint64) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

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
}

// ReadAll liest den gesamten aktuellen Datensatz und gibt die ID zurück.
func (bc *_BngConnChannelByteCache) Read() ([]byte, uint64, error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

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
