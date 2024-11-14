package bngsocket

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// Wird verwendet um RPC Anfragen zu verarbeiten
func (o *BngConn) processRpcRequest(rpcReq *RpcRequest) error {
	// Es wird geprüft ob die gesuchte Zielfunktion vorhanden ist
	var found bool
	var fn reflect.Value
	if rpcReq.Hidden {
		fn, found = o.hiddenFunctions.Load(rpcReq.Name)
	} else {
		fn, found = o.functions.Load(rpcReq.Name)
	}
	if !found {
		return fmt.Errorf("bngsocket->processRpcRequest[0]: unkown function: %s", rpcReq.Name)
	}

	// Context erstellen und an die Funktion übergeben
	ctx := &BngRequest{Conn: o}

	// Es wird versucht die Akommenden Funktionsargumente in den Richtigen Datentypen zu unterteilen
	in, err := convertRPCCallParameterBackToGoValues(o, fn, ctx, rpcReq.Params...)
	if err != nil {
		return fmt.Errorf("processRpcRequest[1]: " + err.Error())
	}

	// Methode PANIC Sicher ausführen ausführen
	results, err := func() (results []reflect.Value, err error) {
		// Defer a function to recover from panic
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("bngsocket->processRpcRequest[2]: panic occurred: %v", r)
				results = nil
			}
		}()

		// Die Funktion wird mittels Reflection aufgerufen
		results = fn.Call(in)

		// Das Ergebniss wird zurückgegeben
		return results, nil
	}()
	if err != nil {
		return fmt.Errorf("bngsocket->processRpcRequest[3]: " + err.Error())
	}

	// Es muss mindestens 1 Eintrag vorhanden sein,
	if len(results) < 1 {
		return fmt.Errorf("return need more the zero values")
	}

	// Der Letzte Eintrag muss ein Error sein
	lasteElementOnResultsArray := results[len(results)-1]
	if lasteElementOnResultsArray.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		// Nun prüfe, ob der Fehler tatsächlich nil ist oder nicht
		if !lasteElementOnResultsArray.IsNil() {
			// Der Fehler wird zurückgesendet
			if err := socketWriteRpcErrorResponse(o, lasteElementOnResultsArray.String(), rpcReq.Id); err != nil {
				return fmt.Errorf("bngsocket->processRpcRequest: " + err.Error())
			}
		}
	}

	// Die Rückgabewerte werden nacheinander abgearbeitet
	// der Letzte Eintrag im Results Array wird ausgelassen.
	values := make([]interface{}, 0)
	for i := range len(results) - 1 {
		values = append(values, results[i].Interface())
	}

	// Es wird geprüft ob die Rückgabewerte zulässig und korrekt sind

	// Die Daten werden für den Transport vorbereitet
	preparedValues, err := processRpcGoDataTypeTransportable(o, values...)
	if err != nil {
		return fmt.Errorf("processRpcRequest: " + err.Error())
	}

	// Die Antwort wird zurückgesendet
	if err := socketWriteRpcSuccessResponse(o, preparedValues, rpcReq.Id); err != nil {
		return fmt.Errorf("processRpcRequest: " + err.Error())
	}

	// Die Antwort wurde erfolgreich zurückgewsendet
	return nil
}

// Wird verwendet um ein RPC Response entgegenzunehmen
func (o *BngConn) processRpcResponse(rpcResp *RpcResponse) error {
	// Es wird geprüft ob es eine Offene Sitzung gibt
	session, found := o.openRpcRequests.Load(rpcResp.Id)
	if !found {
		return fmt.Errorf("bngsocket->processRpcResponse[0]: unkown rpc request session")
	}

	// Wird verwenet um die Antwort in den Cahn zu schreiben
	err := func(rpcResp *RpcResponse) (err error) {
		defer func() {
			if r := recover(); r != nil {
				// Wandelt den Panic-Wert in einen error um
				err = fmt.Errorf("bngsocket->processRpcResponse[1]: session panicked: %v", r)
			}
		}()

		session <- rpcResp

		return nil
	}(rpcResp)
	if err != nil {
		return fmt.Errorf("bngsocket->processRpcResponse[2]: " + err.Error())
	}

	// Kein Fehler aufgetreten
	return nil
}

// Registriert eine Funktion im allgemeien
func (s *BngConn) _RegisterFunction(hidden bool, nameorid string, fn interface{}) error {
	// Es wird geprüft ob die Verbindung getrennt wurde
	if connectionIsClosed(s) {
		return io.EOF
	}

	// Die RPC Funktion wird validiert
	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()
	if err := validateRPCFunction(fnValue, fnType, true); err != nil {
		return fmt.Errorf("bngsocket->_RegisterFunction[0]: " + err.Error())
	}

	// Der connMutextex wird angewendet
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

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
	// Es wird geprüft ob die Verbindung getrennt wurde
	if connectionIsClosed(s) {
		return nil, io.EOF
	}

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
		if connectionIsClosed(s) {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("bngsocket->_CallFunction: " + err.Error())
	}

	// Es wird auf die Antwort gewartet
	response := <-responseChan

	// Es wird geprüft ob die Verbindung getrennt wurde
	if connectionIsClosed(s) {
		return nil, io.EOF
	}

	// Die Requestssitzung wird entfernt
	s.openRpcRequests.Delete(rpcreq.Id)

	// Der Chan wird vollständig geschlossen
	close(responseChan)

	// Es wird geprüft ob ein Fehler vorhanden ist
	if response.Error != "" {
		// Es wird geprüft ob die Verbindung getrennt wurde
		if connectionIsClosed(s) {
			return nil, io.EOF
		}

		// Der Fehler wird zurückgegeben
		return nil, fmt.Errorf(response.Error)
	}

	// Es wird geprüft ob ein Rückgabewert vorhanden ist
	if response.Return != nil {
		// Es wird geprüft ob die Verbindung getrennt wurde
		if connectionIsClosed(s) {
			return nil, io.EOF
		}

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
