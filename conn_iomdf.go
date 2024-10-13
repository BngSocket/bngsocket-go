package bngsocket

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// Nimmt eintreffende Daten entgegen
func (o *BngConn) _ProcessReadedData(data []byte) {
	// Dynamisches Unmarshallen in eine map[string]interface{} oder interface{}
	var typeInfo TypeInfo
	err := msgpack.Unmarshal(data, &typeInfo)
	if err != nil {
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[0]: " + err.Error()))

		// Wird beendet
		return
	}

	// Dynamische Verarbeitung basierend auf dem Typ des Wertes
	switch typeInfo.Type {
	// RPC Pakete
	case "rpcreq", "rpcres":
		switch typeInfo.Type {
		case "rpcreq":
			// Der Datensatz wird als RPC Regquest eingelesen
			var rpcRequest *RpcRequest
			err := msgpack.Unmarshal(data, &rpcRequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[1]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o.processRpcRequest(rpcRequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[2]: " + err.Error()))

				// Wird beendet
				return
			}
		case "rpcres":
			// Der Datensatz wird als RPC Regquest eingelesen
			var rpcResponse *RpcResponse
			err := msgpack.Unmarshal(data, &rpcResponse)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[3]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o.processRpcResponse(rpcResponse); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[4]: " + err.Error()))

				// Wird beendet
				return
			}
		}
	// Channel Pakete
	case "chreq", "chreqresp", "chst", "chsig", "chtsr":
		switch typeInfo.Type {
		case "chreq":
			// Der Datensatz wird ChannelRequest eingelesen
			var channlrequest *ChannelRequest
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[3]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelRequestPackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[4]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chreqresp":
			// Der Datensatz wird als ChannelRequestResponse eingelesen
			var channlrequest *ChannelRequestResponse
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[5]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelRequestResponsePackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[6]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chst":
			// Der Datensatz wird als ChannelSessionDataTransport eingelesen
			var channlrequest *ChannelSessionDataTransport
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[7]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelSessionPackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[8]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chsig":
			// Der Datensatz wird als ChannelSessionTransportSignal eingelesen
			var channlrequest *ChannlSessionTransportSignal
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[9]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelSessionSignal(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[10]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chtsr":
			// Der Datensatz wird als ChannelTransportStateResponse eingelesen
			var channlrequest *ChannelTransportStateResponse
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[11]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelTransportStateResponsePackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[12]: " + err.Error()))

				// Wird beendet
				return
			}
		}
	// Unbekannter Pakettyp
	default:
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[13]: unkown type"))

		// Wird beendet
		return
	}
}

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
	// Der Mutex wird verwendet
	s.mu.Lock()
	defer s.mu.Unlock()

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
	// Der Mutex wird verwendet
	s.mu.Lock()
	defer s.mu.Unlock()

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
	// Der Mutex wird verwendet
	s.mu.Lock()
	defer s.mu.Unlock()

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

// Wird verwendet wenn ein Abweichender Protokoll Fehler auftritt
func (o *BngConn) _ConsensusProtocolTermination(reason error) {
	// Der Mutex wird angewenet
	o.mu.Lock()
	defer o.mu.Unlock()

	// Es wird geprüft ob beretis ein Fehler vorhanden ist
	if o.runningError.Get() != nil {
		return
	}

	// Der Fehler wird geschrieben
	if reason == nil {
		o.runningError.Set(fmt.Errorf(""))
	} else {
		o.runningError.Set(reason)
	}

	// Es wird Signalisiert dass die Verbindung geschlossen wurde
	o.closed.Set(true)

	// Die Socket Verbindung wird geschlossen
	o.conn.Close()
}

// Wird verwendet um mitzuteilen dass die Verbindung getrennt wurde
func (o *BngConn) _ConsensusConnectionClosedSignal() {
	// Der Mutex wird angewenet
	o.mu.Lock()
	defer o.mu.Unlock()

	// Es wird geprüft ob beretis ein Fehler vorhanden ist
	if o.runningError.Get() != nil {
		return
	}

	// Es wird Signalisiert das der Vorgang beendet wurde
	o.closed.Set(true)

	// Der Socket wird geschlossen
	o.conn.Close()
}

// Registriert eine Funktion im allgemeien
func (s *BngConn) _RegisterFunction(hidden bool, nameorid string, fn interface{}) error {
	// Die RPC Funktion wird validiert
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()
	if err := validateRPCFunction(fnValue, fnType, true); err != nil {
		return fmt.Errorf("bngsocket->_RegisterFunction[0]: " + err.Error())
	}

	// Der Mutex wird angewendet
	s.mu.Lock()
	defer s.mu.Unlock()

	// Die Funktion wird Registriert,
	// Es wird unterschieden zwischen Public und Hidden Funktionen
	if hidden {
		// Es wird geprüft ob es bereits eine Funktion mit dem Namen gibt
		if _, found := s.hiddenFunctions.Load(nameorid); found {
			return fmt.Errorf("bngsocket->_RegisterFunction[1]: function always registrated")
		}

		// Die Funktion wird geschieben
		s.hiddenFunctions.Store(nameorid, fnValue)
	} else {
		// Es wird geprüft ob es bereits eine Funktion mit dem Namen gibt
		if _, found := s.functions.Load(nameorid); found {
			return fmt.Errorf("bngsocket->_RegisterFunction[2]: function always registrated")
		}

		// Die Funktion wird geschieben
		s.functions.Store(nameorid, fnValue)
	}

	// Rückgabe
	return nil
}

// Ruft eine Funktion auf der Gegenseite auf
func (s *BngConn) _CallFunction(hiddencall bool, nameorid string, params []interface{}, returnDataType []reflect.Type) ([]interface{}, error) {
	// Es wird geprüft ob die Verwendeten Parameter Zulässigen Datentypen sind
	if err := validateRpcParamsDatatypes(false, params...); err != nil {
		return nil, err
	}

	// Die Parameter werden umgewandelt
	convertedParams, err := processRpcGoDataTypeTransportable(s, params...)
	if err != nil {
		return nil, fmt.Errorf("bngsocket->_CallFunction[0]: " + err.Error())
	}

	// Es wird ein RpcRequest Paket erstellt
	rpcreq := &RpcRequest{
		Type:   "rpcreq",
		Params: convertedParams,
		Name:   nameorid,
		Hidden: hiddencall,
		Id:     strings.ReplaceAll(uuid.NewString(), "-", ""),
	}

	// Das Paket wird in Bytes umgewandelt
	bytedData, err := msgpack.Marshal(rpcreq)
	if err != nil {
		return nil, fmt.Errorf("bngsocket->_CallFunction[1]: " + err.Error())
	}

	// Der Antwort Chan wird erzeugt
	responseChan := make(chan *RpcResponse)

	// Der Response Chan wird zwischengespeichert
	s.openRpcRequests.Store(rpcreq.Id, responseChan)

	// Das Paket wird gesendet
	if err := writeBytesIntoChan(s, bytedData); err != nil {
		return nil, fmt.Errorf("bngsocket->_CallFunction: " + err.Error())
	}

	// Es wird auf die Antwort gewartet
	response := <-responseChan

	// Die Requestssitzung wird entfernt
	s.openRpcRequests.Delete(rpcreq.Id)

	// Der Chan wird vollständig geschlossen
	close(responseChan)

	// Es wird geprüft ob ein Fehler vorhanden ist
	if response.Error != "" {
		// Der Fehler wird zurückgegeben
		return nil, fmt.Errorf(response.Error)
	}

	// Es wird geprüft ob ein Rückgabewert vorhanden ist
	if response.Return != nil {
		// Es wird geprüft ob die Funktion auf der Aufrufendenseite eine Rückgabe erwartet
		if returnDataType == nil {
			return nil, fmt.Errorf("bngsocket->_CallFunction[2]: wanted return, none, has return")
		}

		// Es müssen Soviele Rückgaben vorhanden sein, wie gefordert wurde
		if len(response.Return) != len(returnDataType) {
			for _, item := range response.Return {
				fmt.Println(item)
			}
			return nil, fmt.Errorf("bngsocket->_CallFunction[2]: invalid function return signature, has %d, need %d", len(response.Return), len(returnDataType))
		}

		// Es werden alle Einträge abgearbeitet
		returnValues := make([]interface{}, 0)
		for i := range response.Return {
			value, err := processRPCCallResponseDataToGoDatatype(response.Return[i], returnDataType[i])
			if err != nil {
				return nil, fmt.Errorf("bngsocket->_CallFunction[3]: " + err.Error())
			}
			returnValues = append(returnValues, value)
		}

		// Die Empfangenen Daten werden zurückgegeben
		return returnValues, nil
	}

	// Es ist kein Fehler Aufgetreten, aber es sind auch keine Daten vorhanden
	return nil, nil
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
