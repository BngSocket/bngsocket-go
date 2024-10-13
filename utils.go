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

// IsValidMessagePackType prüft, ob der gegebene Typ für MessagePack gültig ist
func IsValidMessagePackType(t reflect.Type) bool {
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
		return IsValidMessagePackType(t.Elem())
	case reflect.Map:
		// Prüfen, ob Schlüssel und Wert gültige Typen sind
		return IsValidMessagePackType(t.Key()) && IsValidMessagePackType(t.Elem())
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

// IsErrorType prüft, ob der Typ ein error ist
func IsErrorType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	return t == errorType
}

// ValidateStructFields prüft jedes Feld eines Structs auf MessagePack-Kompatibilität
func ValidateStructFields(t reflect.Type) error {
	// Es wird geprüft ob T NULL ist
	if t == nil {
		return fmt.Errorf("ValidateStructFields[0]: 't' is null, not allowed")
	}

	// Stellen Sie sicher, dass der Typ wirklich ein Struct ist
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("ValidateStructFields[1]: expected struct but got %v", t.Kind())
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
			if err := ValidateStructFields(fieldType); err != nil {
				return fmt.Errorf("ValidateStructFields[1]: invalid type in field 1 %s: %w", field.Name, err)
			}
		} else {
			// Für alle anderen Typen prüfen, ob es ein gültiger MessagePack-Typ ist
			if !IsValidMessagePackType(fieldType) {
				return fmt.Errorf("ValidateStructFields[1]: Field %s has an invalid MessagePack type: %s", field.Name, fieldType)
			}
		}
	}

	return nil
}

// Validiert eine RPC Funktion
func ValidateRPCFunction(fnValue reflect.Value, fnType reflect.Type, isRegisterSide bool) error {
	// Es wird geprüft ob der Typ Parameter NULL ist
	if fnType == nil {
		return fmt.Errorf("ValidateRPCFunction[0]: 'fntype' is NULL, not allowed")
	}

	// Prüfe, ob fn eine Funktion ist
	if fnValue.Kind() != reflect.Func {
		return fmt.Errorf("ValidateRPCFunction[1]: only functions allowed")
	}

	// Definiert die Standardwerte
	var beginAt int

	// Prüfe, ob die Funktion mindestens einen Parameter hat
	if isRegisterSide {
		// Es wird geprüft ob mindestens 1 Parameter vorhaden ist
		if fnType.NumIn() < 1 {
			return fmt.Errorf("ValidateRPCFunction[3]: function require one parameter")
		}

		// Die Startposition wird festgelegt
		beginAt = 1

		// Das Request-Objekt wird als Typ verwendet
		contextType := reflect.TypeOf((*BngRequest)(nil)).Elem() // Der zugrunde liegende Typ von `Request`

		// Es wird geprüft, ob die Funktion überhaupt mindestens einen Parameter hat
		if fnType.NumIn() == 0 {
			return fmt.Errorf("ValidateRPCFunction[4]: the function has no parameters")
		}

		// Der Typ des ersten Parameters wird abgerufen
		firstParam := fnType.In(0)

		// Es wird geprüft, ob der erste Parameter ein Pointer auf den zugrunde liegenden Typ des contextType ist
		if !(firstParam.Kind() == reflect.Ptr && firstParam.Elem().ConvertibleTo(contextType)) {
			return fmt.Errorf("ValidateRPCFunction[4]: the first parameter of function isn't compatible with *Request")
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
				return fmt.Errorf("ValidateRPCFunction[5]: parameter %d is a pointer to an unsupported type: %s", i, param.Elem().Kind())
			}

			// Der Struct-Typ ist nur als Pointer zulässig, daher ist dies erlaubt
			if err := ValidateStructFields(param.Elem()); err != nil {
				return fmt.Errorf("ValidateRPCFunction[6]: parameter %d is a pointer to a struct with unsupported fields: %w", i, err)
			}
		} else if param.Kind() == reflect.Func {
			// Wenn das Feld eine Funktion ist, validiere die RPC-Funktion
			fieldValue := reflect.New(param).Elem() // Dummy-Wert für die Funktion erzeugen
			if err := ValidateRPCFunction(fieldValue, param, !isRegisterSide); err != nil {
				return fmt.Errorf("ValidateRPCFunction[7]: Invalid RPC function in field %d: %s", i, err.Error())
			}
		} else {
			// Prüfen, ob es sich um einen Struct handelt (und dieser kein Pointer ist)
			if param.Kind() == reflect.Struct {
				return fmt.Errorf("ValidateRPCFunction[8]: parameter %d is a struct and must be passed as a pointer", i)
			}

			// Wenn es kein Pointer und kein Struct ist, prüfen wir direkt auf MessagePack-Kompatibilität
			if !IsValidMessagePackType(param) {
				return fmt.Errorf("ValidateRPCFunction[9]: parameter %d has an unsupported type: %s", i, param.Kind())
			}
		}
	}

	// Es wird geprüft ob Mindestens 1 Rückgabewert vorhanden ist
	if fnType.NumOut() < 1 {
		return fmt.Errorf("ValidateRPCFunction[10]: At least 1 return value is required")
	}

	// Es wird geprüft ob der Letze Eintrag ein Fehler ist
	lastItem := fnType.Out(fnType.NumOut() - 1)
	if !IsErrorType(lastItem) {
		return fmt.Errorf("ValidateRPCFunction[11]: if the function only has one return value, it must be of type error")
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
			if err := ValidateStructFields(outType); err != nil {
				return fmt.Errorf("der erste Rückgabewert enthält ungültige MessagePack-Typen: %w", err)
			}
		} else {
			// Prüfen, ob der Typ ein Struct ist
			if !IsValidMessagePackType(outType) {
				return fmt.Errorf("der erste Rückgabewert muss ein zulässiger MessagePack-Typ sein")
			}
		}
	}

	// Es handelt sich um eine zulässige Funktion
	return nil
}

// Überprüft ob die Datentypen Zulässig sind um in einer RPC Funktion verwendet werden zu können
func ValidateDatatypeForRpc(param reflect.Type, isRegisterSide bool) error {
	// Wenn der Parameter ein Pointer ist
	if param.Kind() == reflect.Ptr {
		// Prüfen, ob der zugrunde liegende Typ ein zulässiger MessagePack-Typ ist
		if param.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("is a pointer to an unsupported type: %s", param.Elem().Kind())
		}

		// Der Struct-Typ ist nur als Pointer zulässig, daher ist dies erlaubt
		if err := ValidateStructFields(param.Elem()); err != nil {
			return fmt.Errorf("is a pointer to a struct with unsupported fields: %w", err)
		}
	} else if param.Kind() == reflect.Func {
		// Wenn das Feld eine Funktion ist, validiere die RPC-Funktion
		fieldValue := reflect.New(param).Elem() // Dummy-Wert für die Funktion erzeugen
		if err := ValidateRPCFunction(fieldValue, param, !isRegisterSide); err != nil {
			return fmt.Errorf("invalid RPC function  %s", err.Error())
		}
	} else {
		// Prüfen, ob es sich um einen Struct handelt (und dieser kein Pointer ist)
		if param.Kind() == reflect.Struct {
			return fmt.Errorf("is a struct and must be passed as a pointer")
		}

		// Wenn es kein Pointer und kein Struct ist, prüfen wir direkt auf MessagePack-Kompatibilität
		if !IsValidMessagePackType(param) {
			return fmt.Errorf("has an unsupported type: %s", param.Kind())
		}
	}
	return nil
}

// Wird verwendet um zu überprüfen ob die Verwendeten Parameter für einen RPC Funktionsaufruf unterstützt werden
func ValidateRpcParamsDatatypes(isRegisterSide bool, params ...interface{}) error {
	for i, item := range params {
		if err := ValidateDatatypeForRpc(reflect.TypeOf(item), isRegisterSide); err != nil {
			return fmt.Errorf("%d - %s", i, err.Error())
		}
	}
	return nil
}

// Konvertiert die Parameter eines Funktionsaufrufes
func ProcessRpcGoDataTypeTransportable(socket *BngConn, params ...interface{}) ([]*RpcDataCapsle, error) {
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
			if err := socket._RegisterFunction(true, id, item); err != nil {
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

// Wandelt Daten mittels Angabe eines Refelect Types um
func ProcessGoValueToRelectType(value any, expectedType reflect.Type, socket *BngConn) (reflect.Value, error) {
	val := reflect.ValueOf(value)
	switch expectedType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// Es wird geprüft ob es sich um einen Integerwert handelt
		if expectedType.Kind() != reflect.Int && expectedType.Kind() != reflect.Int8 && expectedType.Kind() != reflect.Int16 && expectedType.Kind() != reflect.Int32 && expectedType.Kind() != reflect.Int64 {
			return reflect.Value{}, fmt.Errorf("invalid integer transmitted 1")
		}

		// Es wird ein Leeres Int zurückgegeben
		return val.Convert(expectedType), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Es wird geprüft ob es sich um einen Integerwert handelt
		if expectedType.Kind() != reflect.Uint && expectedType.Kind() != reflect.Uint8 && expectedType.Kind() != reflect.Uint16 && expectedType.Kind() != reflect.Uint32 && expectedType.Kind() != reflect.Uint64 {
			return reflect.Value{}, fmt.Errorf("invalid integer transmitted 2")
		}

		// Es wird ein Leeres Int zurückgegeben
		return val.Convert(expectedType), nil
	case reflect.Float32, reflect.Float64:
		// Wandelt den Datentypen um
		if expectedType.Kind() != reflect.Float32 && expectedType.Kind() != reflect.Float64 {
			return reflect.Value{}, fmt.Errorf("invalid float transmitted")
		}

		// Es wird ein Leeres Int zurückgegeben
		return val.Convert(expectedType), nil
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

// Wandelt RpcDataCapsle zurück in Go Datensätze
func ProcessRpcDataCapsleToGoValue(value *RpcDataCapsle, expectedType reflect.Type, socket *BngConn) (reflect.Value, error) {
	switch expectedType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	case reflect.Float32, reflect.Float64:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	case reflect.Bool:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	case reflect.String:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	case reflect.Slice:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	case reflect.Map:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	case reflect.Ptr:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	case reflect.Func:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	case reflect.Struct:
		return ProcessGoValueToRelectType(value.Value, expectedType, socket)
	default:
		return reflect.Value{}, fmt.Errorf("processValueToGoInterface: unsupoorted datatype")
	}
}

// Konvertiert übertragene Parameter wirder zurück in Go Werte um
func ConvertRPCCallParameterBackToGoValues(socket *BngConn, fn reflect.Value, ctx *BngRequest, params ...*RpcDataCapsle) ([]reflect.Value, error) {
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
		cvalue, err := ProcessRpcDataCapsleToGoValue(param, expectedType, socket)
		if err != nil {
			return nil, fmt.Errorf("ConvertRPCCallParameterBackToGoValues: " + err.Error())
		}

		// Wid zwischengespeichert
		in[i+1] = cvalue
	}

	// Die Rückgabewerte werden zurückgegeben
	return in, nil
}

// Wird verwendet um die Rückgabe Daten eines Aufrufes wieder in Go Datentypen zu Konvertieren
func ProcessRPCCallResponseDataToGoDatatype(rdc *RpcDataCapsle, retunDataType reflect.Type) (interface{}, error) {
	// Es wird ermnittelt um was für einen Typen es sich handelt
	val := reflect.ValueOf(rdc.Value)
	valType := val.Type()

	// Konvertiere den Wert in den exakt erwarteten Typ
	switch {
	case rdc.Type == "int":
		// Es wird geprüft ob es sich um einen Integerwert handelt
		if valType.Kind() != reflect.Int && valType.Kind() != reflect.Int8 && valType.Kind() != reflect.Int16 && valType.Kind() != reflect.Int32 && valType.Kind() != reflect.Int64 {
			return nil, fmt.Errorf("invalid integer transmitted 3")
		}

		// Konvertiere in den exakten Integer-Typ, der erwartet wird
		if val.Kind() >= reflect.Int && val.Kind() <= reflect.Int64 {
			// Konvertiere Ganzzahlen entsprechend dem erwarteten Typ
			return val.Interface(), nil
		}
	case rdc.Type == "uint":
		// Es wird geprüft ob es sich um einen Integerwert handelt
		if valType.Kind() != reflect.Uint && valType.Kind() != reflect.Uint8 && valType.Kind() != reflect.Uint16 && valType.Kind() != reflect.Uint32 && valType.Kind() != reflect.Uint64 {
			// Wenn der Übertragene Datentyp kein Uint ist, wird geprüft ob es sich um ein Integer handelt
			if valType.Kind() != reflect.Int && valType.Kind() != reflect.Int8 && valType.Kind() != reflect.Int16 && valType.Kind() != reflect.Int32 && valType.Kind() != reflect.Int64 {
				return nil, fmt.Errorf("invalid integer transmitted 4")
			}
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

// SplitDataIntoChunks teilt die Daten in Chunks der angegebenen Größe auf
func SplitDataIntoChunks(data []byte, chunkSize int) [][]byte {
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

// Wird verwenet um beim Lessevorgang auf Fehler zu Reagieren
func ReadProcessErrorHandling(socket *BngConn, err error) {
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
func WriteProcessErrorHandling(socket *BngConn, err error) {
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
func RunningBackgroundServingLoop(ipcc *BngConn) bool {
	return !ConnectionIsClosed(ipcc)
}

// Gibt an ob eine Verbinding geschlossen wurde
func ConnectionIsClosed(ipcc *BngConn) bool {
	// Der Mutex wird angewendet
	ipcc.mu.Lock()
	defer ipcc.mu.Unlock()

	// Der Wert wird ermittelt
	value := bool(ipcc.closed.Get() || ipcc.closing.Get() || ipcc.runningError.Get() != nil)

	// Der Wert wird zurückgegeben
	return value
}
