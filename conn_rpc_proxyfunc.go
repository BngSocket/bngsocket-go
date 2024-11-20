package bngsocket

import (
	"fmt"
	"reflect"
)

// proxyHiddenRpcFunction wird als Proxy-Funktion verwendet.
// Diese Funktion erstellt dynamisch eine Funktion basierend auf dem erwarteten Typ,
// um versteckte RPC-Funktionen über Reflection aufzurufen.
//
// Parameter:
//   - s *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - expectedType reflect.Type: Der erwartete Typ der zu erstellenden Funktion.
//   - hiddenFuncId string: Die ID der versteckten Funktion, die aufgerufen werden soll.
//
// Rückgabe:
//   - reflect.Value: Die dynamisch erstellte Funktion, die den erwarteten Typ hat.
func proxyHiddenRpcFunction(s *BngConn, expectedType reflect.Type, hiddenFuncId string) reflect.Value {
	return reflect.MakeFunc(expectedType, func(args []reflect.Value) (results []reflect.Value) {
		// Anzahl der erwarteten Rückgabewerte ermitteln
		numOut := expectedType.NumOut()
		results = make([]reflect.Value, numOut)

		// Die Parameter werden umgewandelt
		params := make([]interface{}, 0)
		for _, item := range args {
			params = append(params, item.Interface())
		}

		// Die Rückgabewerte der Funktionen werden getestet
		reflectTypes := make([]reflect.Type, 0)
		for i := range expectedType.NumOut() {
			reflectTypes = append(reflectTypes, expectedType.Out(i))
		}

		// Fügt eine neue Funktion hinzu
		rpcReturn, callError := _CallFunction(s, true, hiddenFuncId, params, reflectTypes)

		// Rückgabewerte initialisieren
		for i := 0; i < numOut; i++ {
			outType := expectedType.Out(i)
			if outType == reflect.TypeOf((*error)(nil)).Elem() {
				if callError == nil {
					results[i] = reflect.Zero(outType)
				} else {
					results[i] = reflect.ValueOf(rpcReturn)
				}
			} else {
				if rpcReturn != nil {
					v, err := processGoValueToRelectType(rpcReturn, outType, nil)
					if err != nil {
						fmt.Println(err)
					}
					results[i] = reflect.ValueOf(v.Interface())
				} else {
					results[i] = reflect.Zero(outType)
				}
			}
		}

		// Das Ergebniss wird zurückgegeben
		return results
	})
}
