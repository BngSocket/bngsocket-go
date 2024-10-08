package bngsocket

import (
	"fmt"
	"reflect"

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

	// Es muss mindestens 1 Wert in der Rückgabe vorhanden sein
	if len(results) < 1 {
		return fmt.Errorf("bngsocket->processRpcRequest[9]: invalid function call return size")
	}

	// Die Rückgabewerte werden nacheinander abgearbeitet
	var response *RpcResponse
	values := make([]*RpcDataCapsle, 0)
	for i := range results {
		// Es wird geprüft ob es sich um den letzten Eintrag handelt
		if i == len(results)-1 {
			// Prüfen, ob der Rückgabewert ein Fehler ist
			err := results[i].Interface()

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
					return fmt.Errorf("bngsocket->processRpcRequest[4]: error converting errror")
				}

				// Der Fehler wird gebaut
				response = &RpcResponse{
					Id:    rpcReq.Id,
					Type:  "rpcres",
					Error: cerr.Error(),
				}
			}
		} else {
			// Es wird geprüft ob es sich um einen PTR handelt
			if results[i].Kind() == reflect.Ptr {
				// Es muss sich um ein Struct handeln
				if results[i].Elem().Kind() != reflect.Struct {
					return fmt.Errorf("bngsocket->processRpcRequest[6]: only structs as pointer allowed")
				}

				// Die Rückgabewerte werden für den Transport Vorbereitet
				valuet, err := processRpcGoDataTypeTransportable(o, results[i].Interface())
				if err != nil {
					return fmt.Errorf("bngsocket->processRpcRequest[7]: " + err.Error())
				}
				value = valuet[0]
			} else if results[i].Kind() == reflect.Func {
				fmt.Println("RETURN_FUNC")
			} else {
				// Die Rückgabewerte werden für den Transport Vorbereitet
				preparedData, err := processRpcGoDataTypeTransportable(o, results[i].Interface())
				if err != nil {
					return fmt.Errorf("bngsocket->processRpcRequest[8]:  " + err.Error())
				}
				values = append(values, preparedData[0])
			}
		}
	}

	// Die Rückgabe wird in Bytes umgewandelt
	bytedResponse, err := msgpack.Marshal(response)
	if err != nil {
		return fmt.Errorf("bngsocket->processRpcRequest[10]: " + err.Error())
	}

	// Die Daten werden zur bestätigung zurückgesendet
	// Das Paket wird gesendet
	if err := writeBytesIntoChan(o, bytedResponse); err != nil {
		return fmt.Errorf("bngsocket->processRpcRequest[11]: " + err.Error())
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
