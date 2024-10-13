package bngsocket

import (
	"fmt"
	"reflect"
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
	values := make([]*RpcDataCapsle, 0)
	for i := 0; i < len(results)-2; i++ {
		// Es wird geprüft ob es sich um einen PTR handelt
		switch {
		case results[i].Kind() == reflect.Ptr:
			// Es muss sich um ein Struct handeln
			if results[i].Elem().Kind() != reflect.Struct {
				return fmt.Errorf("bngsocket->processRpcRequest[6]: only structs as pointer allowed")
			}

			// Die Rückgabewerte werden für den Transport Vorbereitet
			valuet, err := processRpcGoDataTypeTransportable(o, results[i].Interface())
			if err != nil {
				return fmt.Errorf("bngsocket->processRpcRequest[7]: " + err.Error())
			}
			values = append(values, valuet[0])
		case results[i].Kind() == reflect.Func:
			fmt.Println("RETURN_FUNC")
		default:
			// Die Rückgabewerte werden für den Transport Vorbereitet
			preparedData, err := processRpcGoDataTypeTransportable(o, results[i].Interface())
			if err != nil {
				return fmt.Errorf("bngsocket->processRpcRequest[8]:  " + err.Error())
			}
			values = append(values, preparedData[0])
		}
	}

	// Die Antwort wird zurückgesendet
	if err := socketWriteRpcSuccessResponse(o, values, rpcReq.Id); err != nil {
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
