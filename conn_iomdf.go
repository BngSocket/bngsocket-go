package bngsocket

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// Nimmt eintreffende Daten entgegen
func (o *BngConn) handleReadedData(data []byte) {
	// Dynamisches Unmarshallen in eine map[string]interface{} oder interface{}
	var typeInfo TypeInfo
	err := msgpack.Unmarshal(data, &typeInfo)
	if err != nil {
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[0]: " + err.Error()))

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
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[1]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o.handleRpcRequest(rpcRequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[2]: " + err.Error()))

				// Wird beendet
				return
			}
		case "rpcres":
			// Der Datensatz wird als RPC Regquest eingelesen
			var rpcResponse *RpcResponse
			err := msgpack.Unmarshal(data, &rpcResponse)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[3]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o.handleRpcResponse(rpcResponse); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[4]: " + err.Error()))

				// Wird beendet
				return
			}
		}
	// Channel Pakete
	case "chreq", "chreqresp", "chst", "chsig", "chtsr":
		switch typeInfo.Type {
		case "chreq":
			// Der Datensatz wird als RPC Regquest eingelesen
			var channlrequest *ChannelRequest
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[3]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o.processIncommingChannelRequestPackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[4]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chreqresp":
			// Der Datensatz wird als RPC Regquest eingelesen
			var channlrequest *ChannelRequestResponse
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[5]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o.processIncommingChannelRequestResponsePackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[6]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chst":
			// Der Datensatz wird als RPC Regquest eingelesen
			var channlrequest *ChannelSessionDataTransport
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[7]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o.processIncommingChannelSessionPackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[8]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chsig":
			// Der Datensatz wird als RPC Regquest eingelesen
			var channlrequest *ChannlTransportSignal
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[9]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o.processIncommingChannelClosePackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[10]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chtsr":
			// Der Datensatz wird als RPC Regquest eingelesen
			var channlrequest *ChannelTransportStateResponse
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[11]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o.processIncommingChannelTransportStateResponsePackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[12]: " + err.Error()))

				// Wird beendet
				return
			}
		}
	// Unbekannter Pakettyp
	default:
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		o.consensusProtocolTermination(fmt.Errorf("bngsocket->handleReadedData[13]: unkown type"))

		// Wird beendet
		return
	}
}

// Wird verwendet um RPC Anfragen zu verarbeiten
func (o *BngConn) handleRpcRequest(rpcReq *RpcRequest) error {
	// Es wird geprüft ob die gesuchte Zielfunktion vorhanden ist
	var found bool
	var fn reflect.Value
	if rpcReq.Hidden {
		fn, found = o.hiddenFunctions.Load(rpcReq.Name)
	} else {
		fn, found = o.functions.Load(rpcReq.Name)
	}
	if !found {
		return fmt.Errorf("bngsocket->handleRpcRequest[0]: unkown function: %s", rpcReq.Name)
	}

	// Context erstellen und an die Funktion übergeben
	ctx := &BngRequest{Conn: o}

	// Es wird versucht die Akommenden Funktionsargumente in den Richtigen Datentypen zu unterteilen
	in, err := convertRPCCallParameterBackToGoValues(o, fn, ctx, rpcReq.Params...)
	if err != nil {
		return fmt.Errorf("handleRpcRequest[1]: " + err.Error())
	}

	// Methode PANIC Sicher ausführen ausführen
	results, err := func() (results []reflect.Value, err error) {
		// Defer a function to recover from panic
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("bngsocket->handleRpcRequest[2]: panic occurred: %v", r)
				results = nil
			}
		}()

		// Die Funktion wird mittels Reflection aufgerufen
		results = fn.Call(in)

		// Das Ergebniss wird zurückgegeben
		return results, nil
	}()
	if err != nil {
		return fmt.Errorf("bngsocket->handleRpcRequest[3]: " + err.Error())
	}

	// Die Rückgabewerte werden verarbeitet
	var response *RpcResponse
	if len(results) == 1 {
		// Prüfen, ob der Rückgabewert ein Fehler ist
		err := results[0].Interface()

		// Es wird geprüft ob der Fehler nill ist
		if err == nil {
			// Das Rückgabe Objekt wird erzeugt
			response = &RpcResponse{
				Id:   rpcReq.Id,
				Type: "rpcres",
			}
		} else {
			// Der Fehler wird umgewandelt
			cerr, isok := err.(error)
			if !isok {
				return fmt.Errorf("bngsocket->handleRpcRequest[4]: error converting errror")
			}

			// Der Fehler wird gebaut
			response = &RpcResponse{
				Id:    rpcReq.Id,
				Type:  "rpcres",
				Error: cerr.Error(),
			}
		}
	} else if len(results) == 2 {
		// Die Rückgabewerte werden ausgelesen
		value1, value2 := results[0], results[1].Interface()

		// Es wird geprüft ob ein Fehler vorhanden ist
		if value2 != nil {
			// Es wird geprüft ob es sich um einen Fehler handelt
			err, ok := value2.(error)
			if !ok {
				return fmt.Errorf("bngsocket->handleRpcRequest[5]: error by converting error")
			}

			// Das Rückgabe Objekt wird gebaut
			response = &RpcResponse{
				Type:  "rpcres",
				Id:    rpcReq.Id,
				Error: err.Error(),
			}
		} else {
			// Es wird geprüft ob es sich um einen PTR handelt
			var value *RpcDataCapsle
			if value1.Kind() == reflect.Ptr {
				// Es muss sich um ein Struct handeln
				if value1.Elem().Kind() != reflect.Struct {
					return fmt.Errorf("bngsocket->handleRpcRequest[6]: only structs as pointer allowed")
				}

				// Die Rückgabewerte werden für den Transport Vorbereitet
				valuet, err := processRpcGoDataTypeTransportable(o, value1.Interface())
				if err != nil {
					return fmt.Errorf("bngsocket->handleRpcRequest[7]: " + err.Error())
				}

				value = valuet[0]
			} else if value1.Kind() == reflect.Func {
				fmt.Println("RETURN_FUNC")
			} else {
				// Die Rückgabewerte werden für den Transport Vorbereitet
				preparedData, err := processRpcGoDataTypeTransportable(o, value1.Interface())
				if err != nil {
					return fmt.Errorf("bngsocket->handleRpcRequest[8]:  " + err.Error())
				}

				value = preparedData[0]
			}

			// Das Rückgabe Objekt wird gebaut
			response = &RpcResponse{
				Type:   "rpcres",
				Id:     rpcReq.Id,
				Return: value,
			}
		}
	} else {
		return fmt.Errorf("bngsocket->handleRpcRequest[9]: invalid function call return size")
	}

	// Die Rückgabe wird in Bytes umgewandelt
	bytedResponse, err := msgpack.Marshal(response)
	if err != nil {
		return fmt.Errorf("bngsocket->handleRpcRequest[10]: " + err.Error())
	}

	// Die Daten werden zur bestätigung zurückgesendet
	// Das Paket wird gesendet
	if err := writeBytesIntoChan(o, bytedResponse); err != nil {
		return fmt.Errorf("bngsocket->handleRpcRequest[11]: " + err.Error())
	}

	// Die Antwort wurde erfolgreich zurückgewsendet
	return nil
}

// Wird verwendet um ein RPC Response entgegenzunehmen
func (o *BngConn) handleRpcResponse(rpcResp *RpcResponse) error {
	// Es wird geprüft ob es eine Offene Sitzung gibt
	session, found := o.openRpcRequests.Load(rpcResp.Id)
	if !found {
		return fmt.Errorf("bngsocket->handleRpcResponse[0]: unkown rpc request session")
	}

	// Wird verwenet um die Antwort in den Cahn zu schreiben
	err := func(rpcResp *RpcResponse) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Wandelt den Panic-Wert in einen error um
				err = fmt.Errorf("bngsocket->handleRpcResponse[1]: session panicked: %v", r)
			}
		}()

		session <- rpcResp

		return nil
	}(rpcResp)
	if err != nil {
		return fmt.Errorf("bngsocket->handleRpcResponse[2]: " + err.Error())
	}

	// Kein Fehler aufgetreten
	return nil
}

// Wird verwendet um eintreffende Channel Pakete zu verarbeiten
func (s *BngConn) processIncommingChannelRequestPackage(channlrequest *ChannelRequest) error {
	// Es wird geprüft ob es einen Offenen Listener für die Angefordnerte ID gibt
	channelListener, foundListener := s.openChannelListener.Load(channlrequest.RequestedChannelId)
	if !foundListener {
		// Es wird mitgeteilt dass es sich um einen Unbekannten Channel handelt
		if err := responseUnkownChannel(s, channlrequest.RequestId); err != nil {
			return fmt.Errorf("bngsocket->processIncommingChannelRequestPackage[0]: transmittion error")
		}
		return nil
	}

	// Das Paket wird an den Channel Listener übergeben
	if err := channelListener.processIncommingSessionRequest(channlrequest.RequestId, channlrequest.RequestedChannelId); err != nil {
		return fmt.Errorf("bngsocket->processIncommingChannelRequestPackage[1]: " + err.Error())
	}

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um eintreffende Channel Pakete zu verarbeiten
func (s *BngConn) processIncommingChannelRequestResponsePackage(channlrequest *ChannelRequestResponse) error {
	// Der Mutex wird verwendet
	s.mu.Lock()
	defer s.mu.Unlock()

	// Es wird geprüft ob es einen Offnen Join Vorgang gibt
	joinProcess, foundJoinProcess := s.openChannelJoinProcesses.Load(channlrequest.ReqId)
	if !foundJoinProcess {
		return fmt.Errorf("bngsocket->processIncommingChannelRequestResponsePackage: join process not found")
	}

	// Das Response Paket wird an die Join Funktion zurückgegeben
	joinProcess <- channlrequest

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um eintreffende Channel Pakete zu verarbeiten
func (s *BngConn) processIncommingChannelSessionPackage(channlrequest *ChannelSessionDataTransport) error {
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
			return fmt.Errorf("bngsocket->processIncommingChannelSessionPackage: " + err.Error())
		}

		// Der Vorgang wird ohne Fehler beendet
		return nil
	}

	// Die eingetroffenen Daten werden an den Channel übergeben
	if err := ChannelSessionDataTransport.enterIncommingData(channlrequest.Body, channlrequest.PackageId); err != nil {
		return fmt.Errorf("bngsocket->processIncommingChannelSessionPackage: " + err.Error())
	}

	// Es ist kein Fehler während des Vorgangs aufgetreten
	return nil
}

// Wird verwendet um eintreffende ACK Pakete entgegen zu nehmen
func (s *BngConn) processIncommingChannelTransportStateResponsePackage(channlrequest *ChannelTransportStateResponse) error {
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
			return fmt.Errorf("bngsocket->processIncommingChannelTransportStateResponsePackage: " + err.Error())
		}

		// Der Vorgang wird ohne Fehler beendet
		return nil
	}

	// Der Status wird an den Channel übergeben
	if err := ChannelSessionDataTransport.enterChannelTransportStateResponseSate(channlrequest.PackageId, channlrequest.State); err != nil {
		return fmt.Errorf("bngsocket->processIncommingChannelTransportStateResponsePackage: " + err.Error())
	}

	// Es ist kein Fehler aufgetreten
	return nil
}

// Wird verwendet um eintreffende ACK Pakete entgegen zu nehmen
func (s *BngConn) processIncommingChannelClosePackage(channlrequest *ChannlTransportSignal) error {
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
			return fmt.Errorf("bngsocket->processIncommingChannelClosePackage: " + err.Error())
		}

		// Der Vorgang wird ohne Fehler beendet
		return nil
	}

	// Der Status wird an den Channel übergeben
	if err := ChannelSessionDataTransport.enterSignal(channlrequest.Signal); err != nil {
		return fmt.Errorf("bngsocket->processIncommingChannelClosePackage: " + err.Error())
	}

	// Es ist kein fehler aufgetreten
	return nil
}

// Wird verwendet wenn ein Abweichender Protokoll Fehler auftritt
func (o *BngConn) consensusProtocolTermination(reason error) {
	// Der Mutex wird angewenet
	o.mu.Lock()
	defer o.mu.Unlock()

	// Es wird geprüft ob beretis ein Fehler vorhanden ist
	if o.runningError != nil {
		return
	}

	// Der Fehler wird geschrieben
	if reason == nil {
		o.runningError = fmt.Errorf("")
	} else {
		o.runningError = reason
	}

	// Es wird Signalisiert dass die Verbindung geschlossen wurde
	o.closed.Set(true)

	// Die Socket Verbindung wird geschlossen
	o.conn.Close()
}

// Wird verwendet um mitzuteilen dass die Verbindung getrennt wurde
func (o *BngConn) consensusConnectionClosedSignal() {
	// Der Mutex wird angewenet
	o.mu.Lock()
	defer o.mu.Unlock()

	// Es wird geprüft ob beretis ein Fehler vorhanden ist
	if o.runningError != nil {
		return
	}

	// Es wird Signalisiert das der Vorgang beendet wurde
	o.closed.Set(true)

	// Der Socket wird geschlossen
	o.conn.Close()
}

// Registriert eine Funktion im allgemeien
func (s *BngConn) registerFunctionRoot(hidden bool, nameorid string, fn interface{}) error {
	// Refelction wird auf 'fn' angewendet
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	// Die RPC Funktion wird validiert
	if err := validateRPCFunction(fnValue, fnType, true); err != nil {
		return fmt.Errorf("bngsocket->registerFunctionRoot[0]: " + err.Error())
	}

	// Der Mutex wird angewendet
	s.mu.Lock()
	defer s.mu.Unlock()

	// Die Funktion wird Registriert,
	// Es wird unterschieden zwischen Public und Hidden Funktionen
	if hidden {
		// Es wird geprüft ob es bereits eine Funktion mit dem Namen gibt
		if _, found := s.hiddenFunctions.Load(nameorid); found {
			return fmt.Errorf("bngsocket->registerFunctionRoot[1]: function always registrated")
		}

		// Die Funktion wird geschieben
		s.hiddenFunctions.Store(nameorid, fnValue)
	} else {
		// Es wird geprüft ob es bereits eine Funktion mit dem Namen gibt
		if _, found := s.functions.Load(nameorid); found {
			return fmt.Errorf("bngsocket->registerFunctionRoot[2]: function always registrated")
		}

		// Die Funktion wird geschieben
		s.functions.Store(nameorid, fnValue)
	}

	// Rückgabe
	return nil
}

// Ruft eine Funktion auf der Gegenseite auf
func (s *BngConn) callFunctionRoot(hiddencall bool, nameorid string, params []interface{}, returnDataType reflect.Type) (interface{}, error) {
	// Die Parameter werden umgewandelt
	convertedParams, err := processRpcGoDataTypeTransportable(s, params...)
	if err != nil {
		return nil, fmt.Errorf("bngsocket->callFunctionRoot[0]: " + err.Error())
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
		return nil, fmt.Errorf("bngsocket->callFunctionRoot[1]: " + err.Error())
	}

	// Der Antwort Chan wird erzeugt
	responseChan := make(chan *RpcResponse)

	// Der Response Chan wird zwischengespeichert
	s.mu.Lock()
	s.openRpcRequests.Store(rpcreq.Id, responseChan)
	s.mu.Unlock()

	// Das Paket wird gesendet
	if err := writeBytesIntoChan(s, bytedData); err != nil {
		return nil, fmt.Errorf("bngsocket->callFunctionRoot: " + err.Error())
	}

	// Es wird auf die Antwort gewartet
	response := <-responseChan

	// Die Requestssitzung wird entfernt
	s.mu.Lock()
	s.openRpcRequests.Delete(rpcreq.Id)
	s.mu.Unlock()

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
			return nil, fmt.Errorf("bngsocket->callFunctionRoot[2]: wanted return, none, has return")
		}

		// Die Rückgabewerte werden eingelesen
		value, err := processRPCCallResponseDataToGoDatatype(response.Return, returnDataType)
		if err != nil {
			return nil, fmt.Errorf("bngsocket->callFunctionRoot[3]: " + err.Error())
		}

		// Die Empfangenen Daten werden zurückgegeben
		return value, nil
	}

	// Es ist kein Fehler Aufgetreten, aber es sind auch keine Daten vorhanden
	return nil, nil
}

// Öffnet eine neue Channel Sitzung
func (s *BngConn) registerNewChannelSession(channelSessionId string) (*BngConnChannel, error) {
	// Der Mutex wird angewendet
	s.mu.Lock()
	defer s.mu.Unlock()

	// Es wird geprüft ob der Channel bereits vorhanden ist
	if _, foundChannel := s.openChannelInstances.Load(channelSessionId); foundChannel {
		return nil, fmt.Errorf("bngsocket->registerNewChannelSession: %s always in map", channelSessionId)
	}

	// Der Channel wird erzeugt
	bngsoc := &BngConnChannel{
		socket:              s,
		sesisonId:           channelSessionId,
		isClosed:            newSafeBool(false),
		waitOfPackageACK:    newSafeBool(false),
		openReaders:         newSafeInt(0),
		currentReadingCache: newSafeBytes(nil),
		bytesDataInCache:    newBngConnChannelByteCache(),
		ackChan:             newSafeAck(),
	}

	// Der Channel wird zwischengespeichert
	s.openChannelInstances.Store(channelSessionId, bngsoc)

	// Das Objekt wird zurückgegeben
	return bngsoc, nil
}
