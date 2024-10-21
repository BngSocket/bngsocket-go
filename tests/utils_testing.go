package tests

import (
	"reflect"
	"testing"

	"github.com/CustodiaJS/bngsocket"
)

type InCaseObject struct {
	Username string `rpc:"username"`
}

type TestObject struct {
	Username string        `rpc:"username"`
	Error    error         `rpc:"error"`
	Test     *InCaseObject `rpc:"test"`
}

func testIsErrorType(t *testing.T) {
	if !bngsocket.IsErrorType(reflect.TypeFor[error]()) {
		t.Fatal("IsErrorType is broken, commitet value was error")
	}
	if bngsocket.IsErrorType(reflect.TypeFor[int]()) {
		t.Fatal("IsErrorType is broken, commitet value wasnt error")
	}
}

func testIsValidMessagePackType(t *testing.T) {
	// Es wird geprüft wie die Funktion auf Fehler reagiert
	t.Log("Test nill value without reflecting")
	err := bngsocket.IsValidMessagePackType(nil)
	if err == nil {
		t.Log("Broken IsValidMessagePackType function, accept nill values without reflecting")
	}
	if err.Error() != "passed Value was nil, not allowed" {
		t.Log("Broken IsValidMessagePackType function, accept nill values without reflecting")
	}

	// Die Einfachen Zulässigen Datentypen werden gepürft
	t.Log("Test simple datatypes")
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[int]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[int8]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[int16]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[int32]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[int64]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[uint]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[uint8]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[uint16]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[uint32]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[uint64]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[float32]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[float64]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[bool]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[string]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[byte]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[TestObject]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[error]()); err != nil {
		t.Fatal(err)
	}

	// Die Zulässigen Maps werden gepürft
	t.Log("Test maps")
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[int]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[int8]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[int16]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[int32]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[int64]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[uint]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[uint16]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[uint32]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[uint64]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[float32]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[string]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[map[string]TestObject]()); err != nil {
		t.Fatal(err)
	}

	// Die Zulässigen Slices werden geprüft
	t.Log("Test slices")
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]int]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]int8]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]int16]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]int32]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]int64]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]uint]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]uint8]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]uint16]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]uint32]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]uint64]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]float32]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]float64]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]bool]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]string]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]byte]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]TestObject]()); err != nil {
		t.Fatal(err)
	}
	if err := bngsocket.IsValidMessagePackType(reflect.TypeFor[[]error]()); err != nil {
		t.Fatal(err)
	}
}

func testValidateRPCFunction(t *testing.T) {
	// Es wird eine Zulässige Testfunktion auf der Registrierer Seite
	correctFunctionOnRegisterSide := func(req *bngsocket.BngRequest) error {
		return nil
	}
	value := reflect.ValueOf(correctFunctionOnRegisterSide)
	typeof := value.Type()
	if err := bngsocket.ValidateRPCFunction(value, typeof, true); err != nil {
		t.Fatal(err.Error())
	}

	// Es wird eine nicht Zulässige Testfunktion auf der Register Seite registriert
}

func utilsTesting(t *testing.T) {
	// Die Funktion "IsErrorType" wird getestet
	t.Log("Test 'IsErrorType' function")
	testIsErrorType(t)

	// Die Funktion "IsValidMessagePackType" wird getestet
	t.Log("Test 'IsValidMessagePackType' function")
	testIsValidMessagePackType(t)

	// Die Funktion "ValidateRPCFunction" wird getestet
	t.Log("Test 'ValidateRPCFunction' function")
	testValidateRPCFunction(t)
}
