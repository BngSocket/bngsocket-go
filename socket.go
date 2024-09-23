package bngsocket

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// RegisterFunction ermöglicht es, neue Funktionen dynamisch hinzuzufügen
func (s *BngSocket) RegisterFunction(name string, fn interface{}) error {
	// Fügt eine neue Funktion hinzu
	if err := s.registerFunctionRoot(false, name, fn); err != nil {
		return fmt.Errorf("BngSocket->RegisterFunction[0]: " + err.Error())
	}

	// Kein Fehler aufgetreten
	return nil
}

// CallFunction ruft eine Funktion auf der Gegenseite auf
func (s *BngSocket) CallFunction(name string, params []interface{}, returnDataType reflect.Type) (interface{}, error) {
	// Die Funktion auf der Gegenseite wird aufgerufen
	data, err := s.callFunctionRoot(false, name, params, returnDataType)
	if err != nil {
		return nil, fmt.Errorf("BngSocket->CallFunction[0]: " + err.Error())
	}

	// Kein Fehler aufgetreten
	return data, nil
}

// Wird verwendet um die Verbindung zu schließen
func (s *BngSocket) Close() error {
	// Es wird Markiert dass der Socket geschlossen ist
	s.mu.Lock()
	s.closing = true

	// Der Socket wird geschlossen
	closeerr := s.conn.Close()
	s.mu.Unlock()

	// Es wird gewartet dass alle Hintergrundaufgaben abgeschlossen werden
	s.bp.Wait()

	// Der Writeable Chan wird geschlossen
	close(s.writeableData)

	// Sollte ein Fehler vorhanden sein, wird dieser Zurückgegeben
	if closeerr != nil {
		return fmt.Errorf("BngSocket->Close: " + closeerr.Error())
	}

	// Es ist kein Fehler vorhanden
	return nil
}

// LocalAddr returns the local network address, if known.
func (s *BngSocket) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

// RemoteAddr returns the remote network address, if known.
func (s *BngSocket) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

// SetDeadline sets the read and write deadlines associated with the connection. It is equivalent to calling both SetReadDeadline and SetWriteDeadline.
func (s *BngSocket) SetDeadline(t time.Time) error {
	return s.conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls and any currently-blocked Read call. A zero value for t means Read will not time out.
func (s *BngSocket) SetReadDeadline(t time.Time) error {
	return s.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls and any currently-blocked Write call. Even if write times out, it may return n > 0, indicating that some of the data was successfully written. A zero value for t means Write will not time out.
func (s *BngSocket) SetWriteDeadline(t time.Time) error {
	return s.conn.SetWriteDeadline(t)
}

// Nimmt eintreffende Daten entgegen
func (o *BngSocket) handleReadedData(data []byte) {
	// Dynamisches Unmarshallen in eine map[string]interface{} oder interface{}
	var typeInfo TypeInfo
	err := msgpack.Unmarshal(data, &typeInfo)
	if err != nil {
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		o.consensusProtocolTermination(fmt.Errorf("BngSocket->handleReadedData[0]: " + err.Error()))

		// Wird beendet
		return
	}

	// Dynamische Verarbeitung basierend auf dem Typ des Wertes
	switch typeInfo.Type {
	case "rpcreq":
		// Der Datensatz wird als RPC Regquest eingelesen
		var rpcRequest *RpcRequest
		err := msgpack.Unmarshal(data, &rpcRequest)
		if err != nil {
			// Aus Sicherheitsgründen wird die Verbindung terminiert
			o.consensusProtocolTermination(fmt.Errorf("BngSocket->handleReadedData[1]: " + err.Error()))

			// Wird beendet
			return
		}

		// Das Paket wird weiterverarbeitet
		if err := o.handleRpcRequest(rpcRequest); err != nil {
			// Aus Sicherheitsgründen wird die Verbindung terminiert
			o.consensusProtocolTermination(fmt.Errorf("BngSocket->handleReadedData[2]: " + err.Error()))

			// Wird beendet
			return
		}
	case "rpcres":
		// Der Datensatz wird als RPC Regquest eingelesen
		var rpcResponse *RpcResponse
		err := msgpack.Unmarshal(data, &rpcResponse)
		if err != nil {
			// Aus Sicherheitsgründen wird die Verbindung terminiert
			o.consensusProtocolTermination(fmt.Errorf("BngSocket->handleReadedData[3]: " + err.Error()))

			// Wird beendet
			return
		}

		// Das Paket wird weiterverarbeitet
		if err := o.handleRpcResponse(rpcResponse); err != nil {
			// Aus Sicherheitsgründen wird die Verbindung terminiert
			o.consensusProtocolTermination(fmt.Errorf("BngSocket->handleReadedData[4]: " + err.Error()))

			// Wird beendet
			return
		}
	default:
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		o.consensusProtocolTermination(fmt.Errorf("BngSocket->handleReadedData[4]: unkown type"))

		// Wird beendet
		return
	}
}

// Wird verwendet um RPC Anfragen zu verarbeiten
func (o *BngSocket) handleRpcRequest(rpcReq *RpcRequest) error {
	// Es wird geprüft ob die gesuchte Zielfunktion vorhanden ist
	o.mu.Lock()
	var found bool
	var fn reflect.Value
	if rpcReq.Hidden {
		fn, found = o.hiddenFunctions[rpcReq.Name]
	} else {
		fn, found = o.functions[rpcReq.Name]
	}
	o.mu.Unlock()
	if !found {
		return fmt.Errorf("BngSocket->handleRpcRequest[0]: unkown function: %s", rpcReq.Name)
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
				err = fmt.Errorf("BngSocket->handleRpcRequest[2]: panic occurred: %v", r)
				results = nil
			}
		}()

		// Die Funktion wird mittels Reflection aufgerufen
		results = fn.Call(in)

		// Das Ergebniss wird zurückgegeben
		return results, nil
	}()
	if err != nil {
		return fmt.Errorf("BngSocket->handleRpcRequest[3]: " + err.Error())
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
				return fmt.Errorf("BngSocket->handleRpcRequest[4]: error converting errror")
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
				return fmt.Errorf("BngSocket->handleRpcRequest[5]: error by converting error")
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
					return fmt.Errorf("BngSocket->handleRpcRequest[6]: only structs as pointer allowed")
				}

				// Die Rückgabewerte werden für den Transport Vorbereitet
				valuet, err := processRpcGoDataTypeTransportable(o, value1.Interface())
				if err != nil {
					return fmt.Errorf("BngSocket->handleRpcRequest[7]: " + err.Error())
				}

				value = valuet[0]
			} else if value1.Kind() == reflect.Func {
				fmt.Println("RETURN_FUNC")
			} else {
				// Die Rückgabewerte werden für den Transport Vorbereitet
				preparedData, err := processRpcGoDataTypeTransportable(o, value1.Interface())
				if err != nil {
					return fmt.Errorf("BngSocket->handleRpcRequest[8]:  " + err.Error())
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
		return fmt.Errorf("BngSocket->handleRpcRequest[9]: invalid function call return size")
	}

	// Die Rückgabe wird in Bytes umgewandelt
	bytedResponse, err := msgpack.Marshal(response)
	if err != nil {
		return fmt.Errorf("BngSocket->handleRpcRequest[10]: " + err.Error())
	}

	// Die Daten werden zur bestätigung zurückgesendet
	o.writeableData <- bytedResponse

	// Die Antwort wurde erfolgreich zurückgewsendet
	return nil
}

// Wird verwendet um ein RPC Response entgegenzunehmen
func (o *BngSocket) handleRpcResponse(rpcResp *RpcResponse) error {
	// Es wird geprüft ob es eine Offene Sitzung gibt
	o.mu.Lock()
	session, found := o.openRpcRequests[rpcResp.Id]
	o.mu.Unlock()
	if !found {
		return fmt.Errorf("BngSocket->handleRpcResponse[0]: unkown rpc request session")
	}

	// Wird verwenet um die Antwort in den Cahn zu schreiben
	err := func(rpcResp *RpcResponse) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Wandelt den Panic-Wert in einen error um
				err = fmt.Errorf("BngSocket->handleRpcResponse[1]: session panicked: %v", r)
			}
		}()

		session <- rpcResp

		return nil
	}(rpcResp)
	if err != nil {
		return fmt.Errorf("BngSocket->handleRpcResponse[2]: " + err.Error())
	}

	// Kein Fehler aufgetreten
	return nil
}

// Wird verwendet wenn ein Abweichender Protokoll Fehler auftritt
func (o *BngSocket) consensusProtocolTermination(reason error) {
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
	o.closed = true

	// Die Socket Verbindung wird geschlossen
	o.conn.Close()
}

// Wird verwendet um mitzuteilen dass die Verbindung getrennt wurde
func (o *BngSocket) consensusConnectionClosedSignal() {
	// Der Mutex wird angewenet
	o.mu.Lock()
	defer o.mu.Unlock()

	// Es wird geprüft ob beretis ein Fehler vorhanden ist
	if o.runningError != nil {
		return
	}

	// Es wird Signalisiert das der Vorgang beendet wurde
	o.closed = true

	// Der Socket wird geschlossen
	o.conn.Close()
}

// Registriert eine Funktion im allgemeien
func (s *BngSocket) registerFunctionRoot(hidden bool, nameorid string, fn interface{}) error {
	// Refelction wird auf 'fn' angewendet
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	// Die RPC Funktion wird validiert
	if err := validateRPCFunction(fnValue, fnType, true); err != nil {
		return fmt.Errorf("BngSocket->registerFunctionRoot[0]: " + err.Error())
	}

	// Der Mutex wird angewendet
	s.mu.Lock()
	defer s.mu.Unlock()

	// Die Funktion wird Registriert,
	// Es wird unterschieden zwischen Public und Hidden Funktionen
	if hidden {
		// Es wird geprüft ob es bereits eine Funktion mit dem Namen gibt
		if _, found := s.hiddenFunctions[nameorid]; found {
			return fmt.Errorf("BngSocket->registerFunctionRoot[1]: function always registrated")
		}

		// Die Funktion wird geschieben
		s.hiddenFunctions[nameorid] = fnValue
	} else {
		// Es wird geprüft ob es bereits eine Funktion mit dem Namen gibt
		if _, found := s.functions[nameorid]; found {
			return fmt.Errorf("BngSocket->registerFunctionRoot[2]: function always registrated")
		}

		// Die Funktion wird geschieben
		s.functions[nameorid] = fnValue
	}

	// Rückgabe
	return nil
}

// Ruft eine Funktion auf der Gegenseite auf
func (s *BngSocket) callFunctionRoot(hiddencall bool, nameorid string, params []interface{}, returnDataType reflect.Type) (interface{}, error) {
	// Die Parameter werden umgewandelt
	convertedParams, err := processRpcGoDataTypeTransportable(s, params...)
	if err != nil {
		return nil, fmt.Errorf("BngSocket->callFunctionRoot[0]: " + err.Error())
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
		return nil, fmt.Errorf("BngSocket->callFunctionRoot[1]: " + err.Error())
	}

	// Der Antwort Chan wird erzeugt
	responseChan := make(chan *RpcResponse)

	// Der Response Chan wird zwischengespeichert
	s.mu.Lock()
	s.openRpcRequests[rpcreq.Id] = responseChan
	s.mu.Unlock()

	// Das Paket wird gesendet
	s.writeableData <- bytedData

	// Es wird auf die Antwort gewartet
	response := <-responseChan

	// Die Requestssitzung wird entfernt
	s.mu.Lock()
	delete(s.openRpcRequests, rpcreq.Id)
	s.mu.Unlock()

	// Es wird geprüft ob ein Fehler vorhanden ist
	if response.Error != "" {
		// Der Fehler wird zurückgegeben
		return nil, fmt.Errorf(response.Error)
	}

	// Es wird geprüft ob ein Rückgabewert vorhanden ist
	if response.Return != nil {
		// Es wird geprüft ob die Funktion auf der Aufrufendenseite eine Rückgabe erwartet
		if returnDataType == nil {
			return nil, fmt.Errorf("BngSocket->callFunctionRoot[2]: wanted return, none, has return")
		}

		// Die Rückgabewerte werden eingelesen
		value, err := processRPCCallResponseDataToGoDatatype(response.Return, returnDataType)
		if err != nil {
			return nil, fmt.Errorf("BngSocket->callFunctionRoot[3]: " + err.Error())
		}

		// Die Empfangenen Daten werden zurückgegeben
		return value, nil
	}

	// Es ist kein Fehler Aufgetreten, aber es sind auch keine Daten vorhanden
	return nil, nil
}
