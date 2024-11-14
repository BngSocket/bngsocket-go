package bngsocket

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

// Wird verwendet um eintreffende Channel Request Pakete zu verarbeiten
func (s *BngConn) _ProcessIncommingChannelRequestPackage(channlrequest *ChannelRequest) error {
	// Es wird geprüft ob es einen Offenen Listener für die Angefordnerte ID gibt
	channelListener, foundListener := s.openChannelListener.Load(channlrequest.RequestedChannelId)
	if !foundListener {
		// Es wird mitgeteilt dass es sich um einen Unbekannten Channel handelt
		if err := responseUnkownChannel(s, channlrequest.RequestId); err != nil {
			if errors.Is(err, io.EOF) {
				return io.EOF
			}
			return fmt.Errorf("bngsocket->_ProcessIncommingChannelRequestPackage[0]: transmittion error")
		}
		return nil
	}

	// Das Paket wird an den Channel Listener übergeben
	if err := channelListener.processIncommingSessionRequest(channlrequest.RequestId, channlrequest.RequestedChannelId); err != nil {
		return fmt.Errorf("bngsocket->_ProcessIncommingChannelRequestPackage[1]: " + err.Error())
	}

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um eintreffende Channel Request Response Pakete zu verarbeiten
func (s *BngConn) _ProcessIncommingChannelRequestResponsePackage(channlrequest *ChannelRequestResponse) error {
	// Der connMutextex wird verwendet
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	// Es wird geprüft ob es einen Offnen Join Vorgang gibt
	joinProcess, foundJoinProcess := s.openChannelJoinProcesses.Load(channlrequest.ReqId)
	if !foundJoinProcess {
		return fmt.Errorf("bngsocket->_ProcessIncommingChannelRequestResponsePackage: join process not found")
	}

	// Das Response Paket wird an die Join Funktion zurückgegeben
	joinProcess <- channlrequest

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um eintreffende Channel Session Data Transport Pakete zu verarbeiten
func (s *BngConn) _ProcessIncommingChannelSessionPackage(channlrequest *ChannelSessionDataTransport) error {
	// Es wird geprüft ob die Verbindung geschlossen wurde
	if s.closed.Get() {
		return io.EOF
	}

	// Es wird geprüft ob es einen offnen Channel gibt,
	// wenn ja wird das Paket an diesen Weitergereicht,
	// wenn es keinen passenden Channel gibt, wird dies der Gegenseite mitgeteilt.
	ChannelSessionDataTransport, foundSession := s.openChannelInstances.Load(channlrequest.ChannelSessionId)
	if !foundSession {
		// Der Gegenseite wird mitgeteilt dass kein Offener Channl gefunden wurde
		if err := responseChannelNotOpen(s, ChannelSessionDataTransport.sesisonId); err != nil {
			return fmt.Errorf("bngsocket->_ProcessIncommingChannelSessionPackage: " + err.Error())
		}

		// Der Vorgang wird ohne Fehler beendet
		return nil
	}

	// Die eingetroffenen Daten werden an den Channel übergeben
	if err := ChannelSessionDataTransport.enterIncommingData(channlrequest.Body, channlrequest.PackageId); err != nil {
		return fmt.Errorf("bngsocket->_ProcessIncommingChannelSessionPackage: " + err.Error())
	}

	// Es ist kein Fehler während des Vorgangs aufgetreten
	return nil
}

// Wird verwendet um eintreffende Übermittlungs Bestätigungen für Channel zu verarbeiten
func (s *BngConn) _ProcessIncommingChannelTransportStateResponsePackage(channlrequest *ChannelTransportStateResponse) error {
	// Der connMutextex wird verwendet
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	// Es wird geprüft ob es einen offnen Channel gibt,
	// wenn ja wird das Paket an diesen Weitergereicht,
	// wenn es keinen passenden Channel gibt, wird dies der Gegenseite mitgeteilt.
	ChannelSessionDataTransport, foundSession := s.openChannelInstances.Load(channlrequest.ChannelSessionId)
	if !foundSession {
		// Der Gegenseite wird mitgeteilt dass kein Offener Channl gefunden wurde
		if err := responseChannelNotOpen(s, ChannelSessionDataTransport.sesisonId); err != nil {
			return fmt.Errorf("bngsocket->_ProcessIncommingChannelTransportStateResponsePackage: " + err.Error())
		}

		// Der Vorgang wird ohne Fehler beendet
		return nil
	}

	// Der Status wird an den Channel übergeben
	if err := ChannelSessionDataTransport.enterChannelTransportStateResponseSate(channlrequest.PackageId, channlrequest.State); err != nil {
		return fmt.Errorf("bngsocket->_ProcessIncommingChannelTransportStateResponsePackage: " + err.Error())
	}

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um eintreffende Signal Pakete entgegen zu nehmen
func (s *BngConn) _ProcessIncommingChannelSessionSignal(channlrequest *ChannlSessionTransportSignal) error {
	// Der connMutextex wird verwendet
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	// Es wird geprüft ob es einen offnen Channel gibt,
	// wenn ja wird das Paket an diesen Weitergereicht,
	// wenn es keinen passenden Channel gibt, wird dies der Gegenseite mitgeteilt.
	channelSessionDataTransport, foundSession := s.openChannelInstances.Load(channlrequest.ChannelSessionId)
	if !foundSession {
		// Der Gegenseite wird mitgeteilt dass kein Offener Channl gefunden wurde
		if err := responseChannelNotOpen(s, channlrequest.ChannelSessionId); err != nil {
			return fmt.Errorf("bngsocket->_ProcessIncommingChannelSessionSignal: " + err.Error())
		}

		// Der Vorgang wird ohne Fehler beendet
		return nil
	}

	// Der Status wird an den Channel übergeben
	if err := channelSessionDataTransport.enterSignal(channlrequest.Signal); err != nil {
		return fmt.Errorf("bngsocket->_ProcessIncommingChannelSessionSignal: " + err.Error())
	}

	// Es ist kein fehler aufgetreten
	return nil
}

// Öffnet eine neue Channel Sitzung
func (s *BngConn) _RegisterNewChannelSession(channelSessionId string) (*BngConnChannel, error) {
	// Es wird geprüft ob der Channel bereits vorhanden ist
	if _, foundChannel := s.openChannelInstances.Load(channelSessionId); foundChannel {
		return nil, fmt.Errorf("bngsocket->_RegisterNewChannelSession: %s always in map", channelSessionId)
	}

	// Der Channel wird erzeugt
	bngsoc := &BngConnChannel{
		socket:              s,
		sesisonId:           channelSessionId,
		channelRunningError: newSafeValue[error](nil),
		isClosed:            newSafeBool(false),
		waitOfPackageACK:    newSafeBool(false),
		openReaders:         newSafeInt(0),
		openWriters:         newSafeInt(0),
		currentReadingCache: newSafeBytes(nil),
		bytesDataInCache:    newBngConnChannelByteCache(),
		ackChan:             newSafeAck(),
		mu:                  new(sync.Mutex),
	}

	// Der Channel wird zwischengespeichert
	s.openChannelInstances.Store(channelSessionId, bngsoc)

	// Debug
	_DebugPrint(fmt.Sprintf("Register new Channel '%s'", channelSessionId))

	// Das Objekt wird zurückgegeben
	return bngsoc, nil
}

// Schließet eine Channel Sitzung
func (s *BngConn) _UnregisterChannelSession(channelSessionId string) error {
	// Die SessionId wird gelöscht
	s.openChannelInstances.Delete(channelSessionId)

	// Es ist kein Fehler aufgetreten
	return nil
}
