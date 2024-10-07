package bngsocket

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// Wird verwedet um einen neuen Channel bereitzustellen
func (s *BngConn) OpenChannelListener(cahnnelId string) (*BngConnChannelListener, error) {
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

	// Der Listener wird zurückgegeben
	return listener, nil
}

// Wird verwendet um eine Channel zu öffnen und zu verbinden
func (s *BngConn) JoinChannel(channelId string) (*BngConnChannel, error) {
	// Es wird ein RpcRequest Paket erstellt
	chreq := &ChannelRequest{
		Type:               "chreq",
		RequestId:          strings.ReplaceAll(uuid.NewString(), "-", ""),
		RequestedChannelId: channelId,
	}

	// Das Paket wird in Bytes umgewandelt
	bytedData, err := msgpack.Marshal(chreq)
	if err != nil {
		return nil, fmt.Errorf("bngsocket->callFunctionRoot[1]: " + err.Error())
	}

	// Der Antwort Chan wird erzeugt
	responseChan := make(chan *ChannelRequestResponse)

	// Der Response Chan wird zwischengespeichert
	s.openChannelJoinProcesses.Store(chreq.RequestId, responseChan)

	// Das Paket wird gesendet
	if err := writeBytesIntoChan(s, bytedData); err != nil {
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
	channel, err := s.registerNewChannelSession(response.ChannelId)
	if err != nil {
		return nil, fmt.Errorf("bngsocket->JoinChannel: " + chreq.Error)
	}

	// Der Socket Channel wird zurückgegeben
	return channel, nil
}

// RegisterFunction ermöglicht es, neue Funktionen dynamisch hinzuzufügen
func (s *BngConn) RegisterFunction(name string, fn interface{}) error {
	// Fügt eine neue Funktion hinzu
	if err := s.registerFunctionRoot(false, name, fn); err != nil {
		return fmt.Errorf("bngsocket->RegisterFunction[0]: " + err.Error())
	}

	// Kein Fehler aufgetreten
	return nil
}

// CallFunction ruft eine Funktion auf der Gegenseite auf
func (s *BngConn) CallFunction(name string, params []interface{}, returnDataType reflect.Type) (interface{}, error) {
	// Die Funktion auf der Gegenseite wird aufgerufen
	data, err := s.callFunctionRoot(false, name, params, returnDataType)
	if err != nil {
		return nil, fmt.Errorf("bngsocket->CallFunction[0]: " + err.Error())
	}

	// Kein Fehler aufgetreten
	return data, nil
}

// Wird verwendet um die Verbindung zu schließen
func (s *BngConn) Close() error {
	// Es wird Markiert dass der Socket geschlossen ist
	s.mu.Lock()
	s.closing.Set(true)

	// Der Socket wird geschlossen
	closeerr := s.conn.Close()
	s.mu.Unlock()

	// Es wird gewartet dass alle Hintergrundaufgaben abgeschlossen werden
	s.bp.Wait()

	// Der Writeable Chan wird geschlossen
	s.writingChan.Close()

	// Sollte ein Fehler vorhanden sein, wird dieser Zurückgegeben
	if closeerr != nil {
		return fmt.Errorf("bngsocket->Close: " + closeerr.Error())
	}

	// Es ist kein Fehler vorhanden
	return nil
}

// LocalAddr returns the local network address, if known.
func (s *BngConn) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

// RemoteAddr returns the remote network address, if known.
func (s *BngConn) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection. It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
func (s *BngConn) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls and any currently-blocked Read call. A zero value for t means Read will not time out.
func (s *BngConn) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls and any currently-blocked Write call. Even if write times out, it may return n > 0, indicating that some of the data was successfully written. A zero value for t means Write will not time out.
func (s *BngConn) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
}
