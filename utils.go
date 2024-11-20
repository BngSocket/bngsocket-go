package bngsocket

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/CustodiaJS/bngsocket/transport"
	"github.com/fxamacker/cbor/v2"
)

// Speichert alle Zulässigen Transportdatentypen ab
var supportedTypes = map[string]bool{
	"bool":   true,
	"string": true,
	"map":    true,
	"slice":  true,
	"int":    true,
	"uint":   true,
	"float":  true,
	"struct": true,
	"func":   true,
}

// Speichert alle Explizit Verbotenen Datentypen ab
var explicitNotAllowDataTypes = map[reflect.Kind]bool{
	reflect.TypeFor[_ByteCache]().Kind():              true,
	reflect.TypeFor[BngConn]().Kind():                 true,
	reflect.TypeFor[BngRequest]().Kind():              true,
	reflect.TypeFor[bngConnAcceptingRequest]().Kind(): true,
	reflect.TypeFor[BngConnChannelListener]().Kind():  true,
	reflect.TypeFor[BngConnChannel]().Kind():          true,
}

// isErrorType prüft, ob der Typ ein error ist
func isErrorType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	errorType := reflect.TypeOf((*error)(nil)).Elem()
	return t == errorType
}

// isValidMessagePackType prüft, ob der gegebene Typ für MessagePack gültig ist
func isValidMessagePackType(t reflect.Type) error {
	// T dar nicht nill nein
	if t == nil {
		return fmt.Errorf("passed Value was nil, not allowed")
	}

	// Die Eigentliche Funktion wird definiert
	var function func(t reflect.Type, isMapKey bool, isMapValue bool, isSliceValue bool) error
	function = func(t reflect.Type, isMapKey bool, isMapValue bool, isSliceValue bool) error {
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return nil
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return nil
		case reflect.Float32, reflect.Float64:
			return nil
		case reflect.Bool:
			if isMapKey {
				return fmt.Errorf("isnt allowed bool as key")
			}
			return nil
		case reflect.String:
			return nil
		case reflect.Slice:
			if isMapKey {
				return fmt.Errorf("isnt allowed slice as key")
			}
			// Prüfen, ob das Slice-Element ein gültiger Typ ist
			return function(t.Elem(), isMapKey, isMapValue, true)
		case reflect.Map:
			if isMapKey {
				return fmt.Errorf("isnt allowed map as key")
			}

			// Prüfen, ob Schlüssel und Wert gültige Typen sind
			keyErr := function(t.Key(), true, false, isSliceValue)
			if keyErr != nil {
				return fmt.Errorf("invalid map key type: %w", keyErr)
			}
			valueErr := function(t.Elem(), false, true, isSliceValue)
			if valueErr != nil {
				return fmt.Errorf("invalid map value type: %w", valueErr)
			}
			return nil
		case reflect.Struct:
			// Structs als Key sind nicht zulässig
			if isMapKey {
				return fmt.Errorf("isnt allowed struct as key")
			}

			// Es wird geprüft ob es sich um ein Verbotenes Struct handelt
			if found, blocked := explicitNotAllowDataTypes[t.Kind()]; found && blocked {
				return fmt.Errorf("not allowed struct type")
			}

			// Es werden alle Getaggeten Daten geprüft
			foundedFileds := 0
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				tag := field.Tag.Get("rpc")
				if tag == "" {
					continue
				}

				// Es wird geprüft ob es sich um einen Zulässien Datentypen handelt
				if err := function(field.Type, false, isMapValue, isSliceValue); err != nil {
					return err
				}

				// Es wird aufgezählt, wieviele Daten gefunden wurden
				foundedFileds++
			}

			// Es wird geprüft ob ein Feld gefunden wurde
			if foundedFileds < 1 {
				return fmt.Errorf("object %s has no fields", t.Kind())
			}

			// Es handelt sich um ein zulässiges Slice
			return nil
		case reflect.Interface:
			if isMapKey {
				return fmt.Errorf("isnt allowed interface as key")
			}

			// Zulässiges Interface
			return nil
		case reflect.Ptr:
			if isMapKey {
				return fmt.Errorf("isnt allowed pointer as key")
			}
			return function(t.Elem(), false, isMapValue, isSliceValue)
		default:
			// Ungültiger Typ, basierend auf dem Kontext (Map-Key, Map-Value, Slice-Element)
			var context string
			if isMapKey {
				context = "as map key"
			} else if isMapValue {
				context = "as map value"
			} else if isSliceValue {
				context = "as slice element"
			} else {
				context = "as top-level type"
			}
			return errors.New("unsupported type: " + t.Kind().String() + " " + context)
		}
	}

	// Die Eigentliche Prüfung wird durchgeführt
	return function(t, false, false, false)
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
			if err := isValidMessagePackType(param.Elem()); err != nil {
				return fmt.Errorf("validateRPCFunction[6]: parameter %d is a pointer to a struct with unsupported fields: %w", i, err)
			}
		} else {
			// Prüfen, ob es sich um einen Struct handelt (und dieser kein Pointer ist)
			if param.Kind() == reflect.Struct {
				return fmt.Errorf("validateRPCFunction[8]: parameter %d is a struct and must be passed as a pointer", i)
			}

			// Wenn es kein Pointer und kein Struct ist, prüfen wir direkt auf MessagePack-Kompatibilität
			if err := isValidMessagePackType(param); err != nil {
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
			if err := isValidMessagePackType(outType); err != nil {
				return fmt.Errorf("der erste Rückgabewert enthält ungültige MessagePack-Typen: %w", err)
			}
		} else {
			// Prüfen, ob der Typ ein Struct ist
			if err := isValidMessagePackType(outType); err != nil {
				return fmt.Errorf("der erste Rückgabewert muss ein zulässiger MessagePack-Typ sein")
			}
		}
	}

	// Es handelt sich um eine zulässige Funktion
	return nil
}

// Überprüft ob die Datentypen Zulässig sind um in einer RPC Funktion verwendet werden zu können
func validateDatatypeForRpc(param reflect.Type) error {
	// Wenn der Parameter ein Pointer ist
	if param.Kind() == reflect.Ptr {
		// Prüfen, ob der zugrunde liegende Typ ein zulässiger MessagePack-Typ ist
		if param.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("is a pointer to an unsupported type: %s", param.Elem().Kind())
		}

		// Der Struct-Typ ist nur als Pointer zulässig, daher ist dies erlaubt
		if err := isValidMessagePackType(param.Elem()); err != nil {
			return fmt.Errorf("is a pointer to a struct with unsupported fields: %w", err)
		}
	} else {
		// Prüfen, ob es sich um einen Struct handelt (und dieser kein Pointer ist)
		if param.Kind() == reflect.Struct {
			return fmt.Errorf("is a struct and must be passed as a pointer")
		}

		// Wenn es kein Pointer und kein Struct ist, prüfen wir direkt auf MessagePack-Kompatibilität
		if err := isValidMessagePackType(param); err != nil {
			return fmt.Errorf("has an unsupported type: %s", param.Kind())
		}
	}
	return nil
}

// Wird verwendet um zu überprüfen ob die Verwendeten Parameter für einen RPC Funktionsaufruf unterstützt werden
func validateRpcParamsDatatypes(params ...interface{}) error {
	for i, item := range params {
		if err := validateDatatypeForRpc(reflect.TypeOf(item)); err != nil {
			return fmt.Errorf("%d - %s", i, err.Error())
		}
	}
	return nil
}

// Konvertiert die Parameter eines Funktionsaufrufes
func processRpcGoDataTypeTransportable(params ...interface{}) ([]*transport.RpcDataCapsle, error) {
	newItems := make([]*transport.RpcDataCapsle, 0)
	for i, item := range params {
		// Refelction wird auf 'fn' angewendet
		fnValue := reflect.ValueOf(item)
		fnType := fnValue.Type()

		// Der Datentyp wird extrahiert
		switch fnType.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			newItems = append(newItems, &transport.RpcDataCapsle{Type: "int", Value: item})
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			newItems = append(newItems, &transport.RpcDataCapsle{Type: "uint", Value: item})
		case reflect.Float32, reflect.Float64:
			newItems = append(newItems, &transport.RpcDataCapsle{Type: "float", Value: item})
		case reflect.Bool:
			newItems = append(newItems, &transport.RpcDataCapsle{Type: "bool", Value: item})
		case reflect.String:
			newItems = append(newItems, &transport.RpcDataCapsle{Type: "string", Value: item})
		case reflect.Slice:
			newItems = append(newItems, &transport.RpcDataCapsle{Type: "slice", Value: item})
		case reflect.Map:
			newItems = append(newItems, &transport.RpcDataCapsle{Type: "map", Value: item})
		case reflect.Ptr:
			if fnValue.IsNil() {
				newItems = append(newItems, &transport.RpcDataCapsle{Type: fmt.Sprintf("null-struct:%s", fnType.Elem()), Value: nil})
			} else {
				if fnType.Elem().Kind() != reflect.Struct {
					return nil, fmt.Errorf("convertRPCCallParameters: invalid data type on %d, type %s, x", i, fnType.Elem().Name())
				}
				cborconverted, err := cbor.Marshal(fnValue.Interface())
				if err != nil {
					return nil, fmt.Errorf("convertRPCCallParameters: invalid data type on %d, type %s, b", i, fnType.Elem().Name())
				}
				newItems = append(newItems, &transport.RpcDataCapsle{Type: fmt.Sprintf("struct:%s", fnType.Elem()), Value: cborconverted})
			}
		default:
			return nil, fmt.Errorf("convertRPCCallParameters: invalid data type on %d, type %s, a", i, fnType.Kind())
		}
	}
	return newItems, nil
}

// Wandelt Daten mittels Angabe eines Refelect Types um
func processGoValueToRelectType(value any, expectedType reflect.Type) (reflect.Value, error) {
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
func processRpcDataCapsleToGoValue(value *transport.RpcDataCapsle, expectedType reflect.Type) (reflect.Value, error) {
	switch expectedType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return processGoValueToRelectType(value.Value, expectedType)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return processGoValueToRelectType(value.Value, expectedType)
	case reflect.Float32, reflect.Float64:
		return processGoValueToRelectType(value.Value, expectedType)
	case reflect.Bool:
		return processGoValueToRelectType(value.Value, expectedType)
	case reflect.String:
		return processGoValueToRelectType(value.Value, expectedType)
	case reflect.Slice:
		return processGoValueToRelectType(value.Value, expectedType)
	case reflect.Map:
		return processGoValueToRelectType(value.Value, expectedType)
	case reflect.Ptr:
		return processGoValueToRelectType(value.Value, expectedType)
	case reflect.Struct:
		return processGoValueToRelectType(value.Value, expectedType)
	default:
		return reflect.Value{}, fmt.Errorf("processValueToGoInterface: unsupoorted datatype")
	}
}

// Konvertiert übertragene Parameter wirder zurück in Go Werte um
func convertRPCCallParameterBackToGoValues(fn reflect.Value, ctx *BngRequest, params ...*transport.RpcDataCapsle) ([]reflect.Value, error) {
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
		cvalue, err := processRpcDataCapsleToGoValue(param, expectedType)
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
func processRPCCallResponseDataToGoDatatype(rdc *transport.RpcDataCapsle, retunDataType reflect.Type) (interface{}, error) {
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

// Konvertiert einen Go Datentyp in einen Transport Datatype
func processRpcGoDataTypeTransportableDatatype(params []reflect.Type) ([]string, error) {
	newItems := make([]string, 0)
	for i, item := range params {
		// Refelction wird auf 'fn' angewendet

		// Der Datentyp wird extrahiert
		switch item.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			newItems = append(newItems, "int")
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			newItems = append(newItems, "uint")
		case reflect.Float32, reflect.Float64:
			newItems = append(newItems, "float")
		case reflect.Bool:
			newItems = append(newItems, "bool")
		case reflect.String:
			newItems = append(newItems, "string")
		case reflect.Slice:
			newItems = append(newItems, "slice")
		case reflect.Map:
			newItems = append(newItems, "map")
		case reflect.Ptr:
			newItems = append(newItems, fmt.Sprintf("struct:%s", item.Elem()))
		default:
			return nil, fmt.Errorf("convertRPCCallParameters: invalid data type on %d, type %s, a", i, item.Kind())
		}
	}
	return newItems, nil
}
