package bngsocket

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"syscall"

	"github.com/fxamacker/cbor/v2"
	"github.com/google/uuid"
)

// Speichert alle Zulässigen Transportdatentypen dar
var supportedTypes = map[string]bool{
	"bool":       true,
	"string":     true,
	"map":        true,
	"slice":      true,
	"int":        true,
	"uint":       true,
	"float":      true,
	"struct":     true,
	"func":       true,
	"ssl/tls":    true,
	"mutexguard": true,
}

// isValidMessagePackType prüft, ob der gegebene Typ für MessagePack gültig ist
func isValidMessagePackType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	case reflect.Float32, reflect.Float64:
		return true
	case reflect.Bool:
		return true
	case reflect.String:
		return true
	case reflect.Slice:
		// Prüfen, ob das Slice-Element ein gültiger Typ ist
		return isValidMessagePackType(t.Elem())
	case reflect.Map:
		// Prüfen, ob Schlüssel und Wert gültige Typen sind
		return isValidMessagePackType(t.Key()) && isValidMessagePackType(t.Elem())
	case reflect.Struct:
		return true
	case reflect.Interface:
		// MessagePack erlaubt generische Typen, wenn sie interface{} sind
		return true
	default:
		// Andere Typen sind nicht unterstützt
		return false
	}
}

// isErrorType prüft, ob der Typ ein error ist
func isErrorType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	return t == errorType
}

// validateStructFields prüft jedes Feld eines Structs auf MessagePack-Kompatibilität
func validateStructFields(t reflect.Type) error {
	// Es wird geprüft ob T NULL ist
	if t == nil {
		return fmt.Errorf("validateStructFields[0]: 't' is null, not allowed")
	}

	// Stellen Sie sicher, dass der Typ wirklich ein Struct ist
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("validateStructFields[1]: expected struct but got %v", t.Kind())
	}

	// Iteriere über jedes Feld im Struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type

		// Falls das Feld ein Pointer ist, prüfen wir den zugrunde liegenden Typ
		if fieldType.Kind() == reflect.Ptr {
			fieldType = fieldType.Elem()
		}

		// Falls das Feld ein weiteres Struct ist, rekursiv dessen Felder validieren
		if fieldType.Kind() == reflect.Struct {
			if err := validateStructFields(fieldType); err != nil {
				return fmt.Errorf("validateStructFields[1]: invalid type in field 1 %s: %w", field.Name, err)
			}
		} else {
			// Für alle anderen Typen prüfen, ob es ein gültiger MessagePack-Typ ist
			if !isValidMessagePackType(fieldType) {
				return fmt.Errorf("validateStructFields[1]: Field %s has an invalid MessagePack type: %s", field.Name, fieldType)
			}
		}
	}

	return nil
}

// Validiert eine RPC Funktion
func validateRPCFunction(fnValue reflect.Value, fnType reflect.Type, isRegisterSide bool) error {
	// Es wird geprüft ob der Typ Parameter NULL ist
	if fnType == nil {
		return fmt.Errorf("validateRPCFunction[0]: 'fntype' is NULL, not allowed")
	}

	// Prüfe, ob fn eine Funktion ist
	if fnValue.Kind() != reflect.Func {
		return fmt.Errorf("validateRPCFunction[1]: only functions allowed")
	}

	// Definiert die Standardwerte
	var beginAt int

	// Prüfe, ob die Funktion mindestens einen Parameter hat
	if isRegisterSide {
		// Es wird geprüft ob mindestens 1 Parameter vorhaden ist
		if fnType.NumIn() < 1 {
			return fmt.Errorf("validateRPCFunction[3]: function require one parameter")
		}

		// Die Startposition wird festgelegt
		beginAt = 1

		// Das Request-Objekt wird als Typ verwendet
		contextType := reflect.TypeOf((*BngRequest)(nil)).Elem() // Der zugrunde liegende Typ von `Request`

		// Es wird geprüft, ob die Funktion überhaupt mindestens einen Parameter hat
		if fnType.NumIn() == 0 {
			return fmt.Errorf("validateRPCFunction[4]: the function has no parameters")
		}

		// Der Typ des ersten Parameters wird abgerufen
		firstParam := fnType.In(0)

		// Es wird geprüft, ob der erste Parameter ein Pointer auf den zugrunde liegenden Typ des contextType ist
		if !(firstParam.Kind() == reflect.Ptr && firstParam.Elem().ConvertibleTo(contextType)) {
			return fmt.Errorf("validateRPCFunction[4]: the first parameter of function isn't compatible with *Request")
		}
	} else {
		// Die Startposition wird festgelegt
		beginAt = 0
	}

	// Prüfe die restlichen Parameter auf zulässige MessagePack-Typen
	for i := beginAt; i < fnType.NumIn(); i++ {
		param := fnType.In(i)

		// Wenn der Parameter ein Pointer ist
		if param.Kind() == reflect.Ptr {
			// Prüfen, ob der zugrunde liegende Typ ein zulässiger MessagePack-Typ ist
			if param.Elem().Kind() != reflect.Struct {
				return fmt.Errorf("validateRPCFunction[5]: parameter %d is a pointer to an unsupported type: %s", i, param.Elem().Kind())
			}

			// Der Struct-Typ ist nur als Pointer zulässig, daher ist dies erlaubt
			if err := validateStructFields(param.Elem()); err != nil {
				return fmt.Errorf("validateRPCFunction[6]: parameter %d is a pointer to a struct with unsupported fields: %w", i, err)
			}
		} else if param.Kind() == reflect.Func {
			// Wenn das Feld eine Funktion ist, validiere die RPC-Funktion
			fieldValue := reflect.New(param).Elem() // Dummy-Wert für die Funktion erzeugen
			if err := validateRPCFunction(fieldValue, param, !isRegisterSide); err != nil {
				return fmt.Errorf("validateRPCFunction[7]: Invalid RPC function in field %d: %s", i, err.Error())
			}
		} else {
			// Prüfen, ob es sich um einen Struct handelt (und dieser kein Pointer ist)
			if param.Kind() == reflect.Struct {
				return fmt.Errorf("validateRPCFunction[8]: parameter %d is a struct and must be passed as a pointer", i)
			}

			// Wenn es kein Pointer und kein Struct ist, prüfen wir direkt auf MessagePack-Kompatibilität
			if !isValidMessagePackType(param) {
				return fmt.Errorf("validateRPCFunction[9]: parameter %d has an unsupported type: %s", i, param.Kind())
			}
		}
	}

	// Es wird geprüft ob Mindestens 1 Rückgabewert vorhanden ist
	if fnType.NumOut() < 1 {
		return fmt.Errorf("validateRPCFunction[10]: At least 1 return value is required")
	}

	// Es wird geprüft ob der Letze Eintrag ein Fehler ist
	lastItem := fnType.Out(fnType.NumOut() - 1)
	if !isErrorType(lastItem) {
		return fmt.Errorf("validateRPCFunction[11]: if the function only has one return value, it must be of type error")
	}

	// Es werden alle Rückgabewerte Abgearbeitet, bis auf den letzten
	for i := 0; i < fnType.NumOut()-2; i++ {
		outType := fnType.Out(i)
		if outType.Kind() == reflect.Ptr {
			outType := fnType.Out(0).Elem()
			if outType.Kind() != reflect.Struct {
				return fmt.Errorf("only structs as pointer allowed")
			}

			// Wenn es ein Struct ist, rekursiv alle Felder prüfen
			if err := validateStructFields(outType); err != nil {
				return fmt.Errorf("der erste Rückgabewert enthält ungültige MessagePack-Typen: %w", err)
			}
		} else {
			// Prüfen, ob der Typ ein Struct ist
			if !isValidMessagePackType(outType) {
				return fmt.Errorf("der erste Rückgabewert muss ein zulässiger MessagePack-Typ sein")
			}
		}
	}

	// Es handelt sich um eine zulässige Funktion
	return nil
}

// Konvertiert die Parameter eines Funktionsaufrufes
func processRpcGoDataTypeTransportable(socket *BngConn, params ...interface{}) ([]*RpcDataCapsle, error) {
	newItems := make([]*RpcDataCapsle, 0)
	for i, item := range params {
		// Refelction wird auf 'fn' angewendet
		fnValue := reflect.ValueOf(item)
		fnType := fnValue.Type()

		// Der Datentyp wird extrahiert
		switch fnType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			newItems = append(newItems, &RpcDataCapsle{Type: "int", Value: item})
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			newItems = append(newItems, &RpcDataCapsle{Type: "uint", Value: item})
		case reflect.Float32, reflect.Float64:
			newItems = append(newItems, &RpcDataCapsle{Type: "float", Value: item})
		case reflect.Bool:
			newItems = append(newItems, &RpcDataCapsle{Type: "bool", Value: item})
		case reflect.String:
			newItems = append(newItems, &RpcDataCapsle{Type: "string", Value: item})
		case reflect.Slice:
			newItems = append(newItems, &RpcDataCapsle{Type: "slice", Value: item})
		case reflect.Map:
			newItems = append(newItems, &RpcDataCapsle{Type: "map", Value: item})
		case reflect.Ptr:
			if fnValue.IsNil() {
				newItems = append(newItems, &RpcDataCapsle{Type: fmt.Sprintf("null-struct:%s", fnType.Elem()), Value: nil})
			} else {
				if fnType.Elem().Kind() != reflect.Struct {
					return nil, fmt.Errorf("convertRPCCallParameters: invalid data type on %d, type %s, x", i, fnType.Elem().Name())
				}
				cborconverted, err := cbor.Marshal(fnValue.Interface())
				if err != nil {
					return nil, fmt.Errorf("convertRPCCallParameters: invalid data type on %d, type %s, b", i, fnType.Elem().Name())
				}
				newItems = append(newItems, &RpcDataCapsle{Type: fmt.Sprintf("struct:%s", fnType.Elem()), Value: cborconverted})
			}
		case reflect.Func:
			// Es wird versucht die Funktion als Hidden Funktion zu Registrieren
			id := uuid.New().String()
			if err := socket._RegisterFunctionRoot(true, id, item); err != nil {
				return nil, fmt.Errorf("bngsocket->RegisterFunction: " + err.Error())
			}

			// Es wird ein neuer HiddenSharedFunction eintrag erzeugt
			hiddenSharedFunctionLink := &RpcHiddenFunction{
				FunctionId: id,
			}

			// Die Daten werden mittels CBOR umgewandelt
			cborconverted, err := cbor.Marshal(hiddenSharedFunctionLink)
			if err != nil {
				return nil, fmt.Errorf("convertRPCCallParameters: invalid data type on %d, type %s, b", i, fnType.Elem().Name())
			}

			// Es wird ein neuer RpcDataCaplse erzeugt
			newItems = append(newItems, &RpcDataCapsle{Type: "func", Value: cborconverted})
		default:
			return nil, fmt.Errorf("convertRPCCallParameters: invalid data type on %d, type %s, a", i, fnType.Kind())
		}
	}
	return newItems, nil
}

// Wandelt Transportierte Werte in Go Werte um
func processValueToGoValue(value any, expectedType reflect.Type, socket *BngConn) (reflect.Value, error) {
	val := reflect.ValueOf(value)
	switch expectedType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Es wird geprüft ob es sich um einen Integerwert handelt
		if expectedType.Kind() != reflect.Int && expectedType.Kind() != reflect.Int8 && expectedType.Kind() != reflect.Int16 && expectedType.Kind() != reflect.Int32 && expectedType.Kind() != reflect.Int64 {
			return reflect.Value{}, fmt.Errorf("invalid integer transmitted")
		}

		// Konvertiere in den exakten Integer-Typ, der erwartet wird
		if val.Kind() >= reflect.Int && val.Kind() <= reflect.Int64 {
			// Konvertiere Ganzzahlen entsprechend dem erwarteten Typ
			return val.Convert(expectedType), nil
		}

		// Es wird ein Leeres Int zurückgegeben
		return reflect.Zero(expectedType), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Es wird geprüft ob es sich um einen Integerwert handelt
		if expectedType.Kind() != reflect.Uint && expectedType.Kind() != reflect.Uint8 && expectedType.Kind() != reflect.Uint16 && expectedType.Kind() != reflect.Uint32 && expectedType.Kind() != reflect.Uint64 {
			return reflect.Value{}, fmt.Errorf("invalid integer transmitted")
		}

		// Konvertiere in den exakten Integer-Typ, der erwartet wird
		if val.Kind() >= reflect.Uint && val.Kind() <= reflect.Uint64 {
			// Konvertiere Ganzzahlen entsprechend dem erwarteten Typ
			return val.Convert(expectedType), nil
		}

		// Es wird ein Leeres Int zurückgegeben
		return reflect.Zero(expectedType), nil
	case reflect.Float32, reflect.Float64:
		// Wandelt den Datentypen um
		if expectedType.Kind() != reflect.Float32 && expectedType.Kind() != reflect.Float64 {
			return reflect.Value{}, fmt.Errorf("invalid float transmitted")
		}

		// Konvertiere in den exakten Float-Typ, der erwartet wird
		if val.Kind() == reflect.Float32 || val.Kind() == reflect.Float64 {
			return val.Convert(expectedType), nil
		}

		// Es wird ein Leeres Int zurückgegeben
		return reflect.Zero(expectedType), nil
	case reflect.Bool:
		return reflect.ValueOf(value), nil
	case reflect.String:
		return reflect.ValueOf(value), nil
	case reflect.Slice:
		return reflect.ValueOf(value), nil
	case reflect.Map:
		return reflect.ValueOf(value), nil
	case reflect.Ptr:
		// Erstelle einen neuen Zeiger auf das erwartete Struct
		structPtr := reflect.New(expectedType)

		// Annahme: param.Value enthält die CBOR-kodierten Daten als []byte
		cborData, ok := value.([]byte)
		if !ok {
			return reflect.Value{}, fmt.Errorf("erwartete []byte für struct, erhalten: %T", value)
		}

		// Deserialisiere die CBOR-Daten in das Struct
		err := cbor.Unmarshal(cborData, structPtr.Interface())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("fehler beim Deserialisieren von CBOR-Daten: %v", err)
		}

		// Setze den konvertierten Wert in das Eingangsarray
		return structPtr.Elem(), nil
	case reflect.Func:
		// Die Daten werden mittels CBOR eingelesen
		// Annahme: param.Value enthält die CBOR-kodierten Daten als []byte
		cborData, ok := value.([]byte)
		if !ok {
			return reflect.Value{}, fmt.Errorf("erwartete []byte für struct, erhalten: %T", value)
		}

		// Deserialisiere die CBOR-Daten in das Struct
		var rhfunc *RpcHiddenFunction
		err := cbor.Unmarshal(cborData, &rhfunc)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("fehler beim Deserialisieren von CBOR-Daten: %v", err)
		}

		// Die Funktion wird registriert
		return proxyHiddenRpcFunction(socket, expectedType, rhfunc.FunctionId), nil
	case reflect.Struct:
		// Erstelle einen neuen Zeiger auf das erwartete Struct
		structPtr := reflect.New(expectedType)

		// Annahme: param.Value enthält die CBOR-kodierten Daten als []byte
		cborData, ok := value.([]byte)
		if !ok {
			return reflect.Value{}, fmt.Errorf("erwartete []byte für struct, erhalten: %T", value)
		}

		// Deserialisiere die CBOR-Daten in das Struct
		err := cbor.Unmarshal(cborData, structPtr.Interface())
		if err != nil {
			return reflect.Value{}, fmt.Errorf("fehler beim Deserialisieren von CBOR-Daten: %v", err)
		}

		// Setze den konvertierten Wert in das Eingangsarray
		return structPtr.Elem(), nil
	default:
		return reflect.Value{}, fmt.Errorf("processValueToGoInterface: unsupoorted datatype")
	}
}

// Konvertiert übertragene Parameter wirder zurück in Go Werte um
func convertRPCCallParameterBackToGoValues(socket *BngConn, fn reflect.Value, ctx *BngRequest, params ...*RpcDataCapsle) ([]reflect.Value, error) {
	// Parametertypen prüfen und aufbereiten
	in := make([]reflect.Value, len(params)+1)
	in[0] = reflect.ValueOf(ctx)

	// ES wird geprüft ob genau die Anzahl der Parameter vorhanden ist
	if fn.Type().NumIn() != len(params)+1 {
		return nil, fmt.Errorf("transmitted function call hast not same parameters, wanted %d, have %d", fn.Type().NumIn(), len(params)+1)
	}

	// Übergebe die weiteren Parameter an die Funktion
	for i, param := range params {
		// Es wird ermnittelt um was für einen Typen es sich handelt
		expectedType := fn.Type().In(i + 1) // Den erwarteten Typ der Funktion an der Stelle ermittelnconvertRPCCallParameters

		// Es wird geprüft ob es sich um einen Zulässigen Datentyp handelt
		if !supportedTypes[strings.Split(param.Type, ":")[0]] {
			return nil, fmt.Errorf("unsuported datatype %s on %d", param.Type, i)
		}

		// Der Wert wird eingelesen
		cvalue, err := processValueToGoValue(param.Value, expectedType, socket)
		if err != nil {
			return nil, fmt.Errorf("convertRPCCallParameterBackToGoValues: " + err.Error())
		}

		// Wid zwischengespeichert
		in[i+1] = cvalue
	}

	// Die Rückgabewerte werden zurückgegeben
	return in, nil
}

// Wird verwendet um die Rückgabe Daten eines Aufrufes wieder in Go Datentypen zu Konvertieren
func processRPCCallResponseDataToGoDatatype(rdc *RpcDataCapsle, retunDataType reflect.Type) (interface{}, error) {
	// Es wird ermnittelt um was für einen Typen es sich handelt
	val := reflect.ValueOf(rdc.Value)
	valType := val.Type()

	// Konvertiere den Wert in den exakt erwarteten Typ
	switch {
	case rdc.Type == "int":
		// Es wird geprüft ob es sich um einen Integerwert handelt
		if valType.Kind() != reflect.Int && valType.Kind() != reflect.Int8 && valType.Kind() != reflect.Int16 && valType.Kind() != reflect.Int32 && valType.Kind() != reflect.Int64 {
			return nil, fmt.Errorf("invalid integer transmitted")
		}

		// Konvertiere in den exakten Integer-Typ, der erwartet wird
		if val.Kind() >= reflect.Int && val.Kind() <= reflect.Int64 {
			// Konvertiere Ganzzahlen entsprechend dem erwarteten Typ
			return val.Interface(), nil
		}
	case rdc.Type == "uint":
		// Es wird geprüft ob es sich um einen Integerwert handelt
		if valType.Kind() != reflect.Uint && valType.Kind() != reflect.Uint8 && valType.Kind() != reflect.Uint16 && valType.Kind() != reflect.Uint32 && valType.Kind() != reflect.Uint64 {
			return nil, fmt.Errorf("invalid integer transmitted")
		}

		// Konvertiere in den exakten Integer-Typ, der erwartet wird
		if val.Kind() >= reflect.Uint && val.Kind() <= reflect.Uint64 {
			// Konvertiere Ganzzahlen entsprechend dem erwarteten Typ
			return val.Interface(), nil
		}
	case rdc.Type == "float":
		// Wandelt den Datentypen um
		if valType.Kind() != reflect.Float32 && valType.Kind() != reflect.Float64 {
			return nil, fmt.Errorf("invalid float transmitted")
		}

		// Konvertiere in den exakten Float-Typ, der erwartet wird
		if val.Kind() == reflect.Float32 || val.Kind() == reflect.Float64 {
			return val.Interface(), nil
		}
	case strings.Split(rdc.Type, ":")[0] == "struct":
		// Erstelle einen neuen Zeiger auf das erwartete Struct
		structPtr := reflect.New(retunDataType)

		// Annahme: param.Value enthält die CBOR-kodierten Daten als []byte
		cborData, ok := rdc.Value.([]byte)
		if !ok {
			return nil, fmt.Errorf("erwartete []byte für struct, erhalten: %T", rdc.Value)
		}

		// Deserialisiere die CBOR-Daten in das Struct
		err := cbor.Unmarshal(cborData, structPtr.Interface())
		if err != nil {
			return nil, fmt.Errorf("fehler beim Deserialisieren von CBOR-Daten: %v", err)
		}

		// Derefeniere den Zeiger, um den Struct-Wert zu erhalten
		return structPtr.Elem().Interface(), nil
	case rdc.Type == "bool" || rdc.Type == "string" || rdc.Type == "map" || rdc.Type == "slice":
		return val.Interface(), nil
	default:
		return nil, fmt.Errorf("invalid data type on %s", valType.Kind())
	}

	// Due Werte werden zurückgegeben
	return rdc.Value, nil
}

// splitDataIntoChunks teilt die Daten in Chunks der angegebenen Größe auf
func splitDataIntoChunks(data []byte, chunkSize int) [][]byte {
	var chunks [][]byte
	cSize := chunkSize - 3
	for len(data) > 0 {
		if len(data) < cSize {
			cSize = len(data)
		}
		chunks = append(chunks, data[:cSize])
		data = data[cSize:]
	}
	return chunks
}

// Wid als Proxy Funktion verwendet
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
		rpcReturn, callError := s._CallFunction(true, hiddenFuncId, params, reflectTypes)

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
					v, err := processValueToGoValue(rpcReturn, outType, nil)
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

// Wird verwenet um beim Lessevorgang auf Fehler zu Reagieren
func readProcessErrorHandling(socket *BngConn, err error) {
	// Der Fehler wird ermittelt
	if errors.Is(err, io.EOF) {
		// Die Verbindung wurde getrennt (EOF)
		socket._ConsensusConnectionClosedSignal()
		return
	} else if errors.Is(err, syscall.ECONNRESET) {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
		return
	} else if errors.Is(err, syscall.EPIPE) {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
		return
	} else {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantReading: " + err.Error()))
		return
	}
}

// Wird verwenet um beim Lessevorgang auf Fehler zu Reagieren
func writeProcessErrorHandling(socket *BngConn, err error) {
	// Der Fehler wird ermittelt
	if errors.Is(err, io.EOF) {
		// Die Verbindung wurde getrennt (EOF)
		socket._ConsensusConnectionClosedSignal()
		return
	} else if errors.Is(err, syscall.ECONNRESET) {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
		return
	} else if errors.Is(err, syscall.EPIPE) {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
		return
	} else {
		// Verbindung wurde vom Peer zurückgesetzt
		socket._ConsensusProtocolTermination(fmt.Errorf("bngsocket->constantWriting: " + err.Error()))
		return
	}
}

// Gibt an ob die Hintergrund Dauerschleifen eines Sockets aktiv sein sollen
func runningBackgroundServingLoop(ipcc *BngConn) bool {
	return !connectionIsClosed(ipcc)
}

// Gibt an ob eine Verbinding geschlossen wurde
func connectionIsClosed(ipcc *BngConn) bool {
	// Der Mutex wird angewendet
	ipcc.mu.Lock()
	defer ipcc.mu.Unlock()

	// Der Wert wird ermittelt
	value := bool(ipcc.closed.Get() || ipcc.closing.Get() || ipcc.runningError.Get() != nil)

	// Der Wert wird zurückgegeben
	return value
}
