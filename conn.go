package bngsocket

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/custodia-cenv/bngsocket-go/transport"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// OpenChannelListener wird verwendet, um einen neuen Channel bereitzustellen.
// Diese Methode erstellt einen neuen Channel Listener, der auf eingehende Channel-Anfragen lauscht.
// Der Listener wird in der openChannelListener Map gespeichert, um zukünftige Anfragen zu verwalten.
//
// Parameter:
//   - channelId string: Die eindeutige ID des Channels, für den der Listener erstellt werden soll.
//
// Rückgabe:
//   - *BngConnChannelListener: Ein Zeiger auf das neu erstellte Channel Listener Objekt.
//   - error: Ein Fehler, falls beim Erstellen oder Speichern des Listeners ein Problem auftritt, ansonsten nil.
func (s *BngConn) OpenChannelListener(cahnnelId string) (*BngConnChannelListener, error) {
	// Der Objekt Mutex wird verwendet
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	// Es wird ein neur Listener erzeugt und abgespeichert
	listener := &BngConnChannelListener{
		socket:          s,
		mu:              new(sync.Mutex),
		waitOfAccepting: NewSafeChan[*bngConnAcceptingRequest](),
	}

	// Es wird geprüft ob es bereits einen Socket Listener gibt, welcher neue Channel Anfragen Entgegen nimmt
	if _, ok := s.openChannelListener.Load(cahnnelId); ok {
		// Es wird geprüft ob ein Running Error vorhanden ist, wenn ja wird dieser Zurückgegeben
		return nil, fmt.Errorf("bngsocket->OpenChannelListener[0]: has alwasys socket listner with name: %s", cahnnelId)
	}

	// Der Eintrag wird hinzugefügt
	s.openChannelListener.Store(cahnnelId, listener)

	// LOG
	_DebugPrint(fmt.Sprintf("BngConn(%s): New Channel Listener: %s", s._innerhid, cahnnelId))

	// Der Listener wird zurückgegeben
	return listener, nil
}

// JoinChannel wird verwendet, um einem bestehenden Channel beizutreten und eine Verbindung herzustellen.
// Diese Methode sendet eine Join-Anfrage an den Server und wartet auf die Antwort, um die Verbindung zu bestätigen.
//
// Parameter:
//   - channelId string: Die ID des Channels, dem beigetreten werden soll.
//
// Rückgabe:
//   - *BngConnChannel: Ein Zeiger auf das verbundene Channel-Objekt.
//   - error: Ein Fehler, falls beim Beitritt zum Channel ein Problem auftritt, ansonsten nil.
func (s *BngConn) JoinChannel(channelId string) (*BngConnChannel, error) {
	// Es wird ein RpcRequest Paket erstellt
	chreq := &transport.ChannelRequest{
		Type:               "chreq",
		RequestId:          strings.ReplaceAll(uuid.NewString(), "-", ""),
		RequestedChannelId: channelId,
	}

	// Das Paket wird in Bytes umgewandelt
	bytedData, err := msgpack.Marshal(chreq)
	if err != nil {
		return nil, fmt.Errorf("bngsocket->_CallFunction[1]: " + err.Error())
	}

	// Der Antwort Chan wird erzeugt
	responseChan := make(chan *transport.ChannelRequestResponse)

	// Der Response Chan wird zwischengespeichert
	s.openChannelJoinProcesses.Store(chreq.RequestId, responseChan)

	// Das Paket wird gesendet
	if err := writeBytesIntoSocketConn(s, bytedData); err != nil {
		return nil, fmt.Errorf("bngsocket->JoinChannel: " + err.Error())
	}

	// Es wird auf die Antwort gewartet
	response := <-responseChan

	// Die Requestssitzung wird entfernt
	s.openChannelJoinProcesses.Delete(chreq.RequestId)

	// Der Chan wird geschlossen
	close(responseChan)

	// Es wird geprüft ob die Anfrage von der Gegenseite angenommen wurde
	if response.NotAcceptedByReason != "" {
		return nil, fmt.Errorf("bngsocket->JoinChannel[1]: " + response.NotAcceptedByReason)
	}

	// Der Channel Vorgagn wird Registriert
	channel, err := s._RegisterNewChannelSession(response.ChannelId)
	if err != nil {
		return nil, fmt.Errorf("bngsocket->JoinChannel: " + chreq.Error)
	}

	// LOG
	_DebugPrint(fmt.Sprintf("BngConn(%s): Channel Joined: %s", s._innerhid, channel.sesisonId))

	// Dem Server wird mitgeteilt dass die Verbindung erfolgreich zustande gekommen ist
	if err := channelWriteACKForJoin(s, response.ChannelId); err != nil {
		// Die Verbindung wird Entregistriert
		s._UnregisterChannelSession(channelId)
		return nil, err
	}

	// Der Socket Channel wird zurückgegeben
	return channel, nil
}

// RegisterFunction ermöglicht es, neue Funktionen dynamisch hinzuzufügen.
// Diese Methode registriert eine Funktion unter einem bestimmten Namen, sodass sie später über RPC aufgerufen werden kann.
//
// Parameter:
//   - name string: Der eindeutige Name, unter dem die Funktion registriert werden soll.
//   - fn interface{}: Die Funktion, die registriert werden soll. Sie muss eine gültige Funktion sein.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Registrieren der Funktion ein Problem aufgetreten ist, ansonsten nil.
func (s *BngConn) RegisterFunction(name string, fn interface{}) error {
	// Fügt eine neue Funktion hinzu
	if err := _RegisterFunction(s, name, fn); err != nil {
		return fmt.Errorf("bngsocket->RegisterFunction[0]: " + err.Error())
	}

	// Kein Fehler aufgetreten
	return nil
}

// CallFunction ruft eine Funktion auf der Gegenseite (Remote) auf.
// Diese Methode sendet eine Funktionsanfrage über die Socket-Verbindung und wartet auf die Antwort.
//
// Parameter:
//   - name string: Der Name der Funktion, die auf der Gegenseite aufgerufen werden soll.
//   - params []interface{}: Ein Slice von Parametern, die an die Funktion übergeben werden.
//   - returnDataType []reflect.Type: Ein Slice von Rückgabetypen, die die erwarteten Rückgabewerte der Funktion definieren.
//
// Rückgabe:
//   - []interface{}: Ein Slice von Rückgabewerten der aufgerufenen Funktion.
//   - error: Ein Fehler, falls beim Aufrufen der Funktion ein Problem aufgetreten ist, ansonsten nil.
func (s *BngConn) CallFunction(name string, params []interface{}, returnDataType []reflect.Type) ([]interface{}, error) {
	// Die Funktion auf der Gegenseite wird aufgerufen
	data, err := _CallFunction(s, name, params, returnDataType)
	if err != nil {
		return nil, err
	}

	// Kein Fehler aufgetreten
	return data, nil
}

// Close wird verwendet, um die Verbindung zu schließen.
// Diese Methode prüft zunächst, ob die Verbindung bereits geschlossen wurde.
// Falls nicht, wird die Verbindung vollständig geschlossen.
// Bei erfolgreichem Schließen wird nil zurückgegeben, andernfalls der aufgetretene Fehler.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Schließen der Verbindung ein Problem aufgetreten ist, ansonsten nil.
func (s *BngConn) Close() error {
	// Es wird geprüft ob die Verbindung bereits getrennt wurde,
	// wenn ja wird ein Fehler zurückgegeben
	if connectionIsClosed(s) {
		return io.EOF
	}

	// Die Verbindung wird Final geschlossen
	if err := fullCloseConn(s); err != nil {
		return err
	}

	// Es ist kein Fehler vorhanden
	return nil
}

// LocalAddr gibt die lokale Netzwerkadresse zurück, falls bekannt.
// Diese Methode ruft die LocalAddr-Methode der zugrunde liegenden Verbindung auf.
//
// Rückgabe:
//   - net.Addr: Die lokale Netzwerkadresse der Verbindung.
func (s *BngConn) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

// RemoteAddr gibt die entfernte Netzwerkadresse zurück, falls bekannt.
// Diese Methode ruft die RemoteAddr-Methode der zugrunde liegenden Verbindung auf.
//
// Rückgabe:
//   - net.Addr: Die entfernte Netzwerkadresse der Verbindung.
func (s *BngConn) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

// SetDeadline setzt die Lese- und Schreib-Deadlines, die mit der Verbindung verknüpft sind.
// Es ist äquivalent zum gleichzeitigen Aufruf von SetReadDeadline und SetWriteDeadline.
//
// Parameter:
//   - t time.Time: Die Zeit, bis zu der Lese- und Schreiboperationen abgeschlossen sein müssen.
//     Eine Null-Zeit bedeutet, dass keine Deadlines gesetzt werden.
func (s *BngConn) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}

// SetReadDeadline setzt die Deadline für zukünftige Read-Aufrufe und alle aktuell blockierten Read-Aufrufe.
// Eine Null-Zeit für t bedeutet, dass Read-Aufrufe nicht ablaufen.
//
// Parameter:
//   - t time.Time: Die Zeit, bis zu der ein Read-Aufruf abgeschlossen sein muss.
//     Eine Null-Zeit bedeutet, dass keine Deadline gesetzt wird.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Setzen der Deadline ein Problem aufgetreten ist, ansonsten nil.
func (s *BngConn) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

// SetWriteDeadline setzt die Deadline für zukünftige Write-Aufrufe und alle aktuell blockierten Write-Aufrufe.
// Selbst wenn das Schreiben aufgrund der Deadline abläuft, kann es n > 0 zurückgeben, was bedeutet, dass einige Daten erfolgreich geschrieben wurden.
// Eine Null-Zeit für t bedeutet, dass Write-Aufrufe nicht ablaufen.
//
// Parameter:
//   - t time.Time: Die Zeit, bis zu der ein Write-Aufruf abgeschlossen sein muss.
//     Eine Null-Zeit bedeutet, dass keine Deadline gesetzt wird.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Setzen der Deadline ein Problem aufgetreten ist, ansonsten nil.
func (s *BngConn) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
}
