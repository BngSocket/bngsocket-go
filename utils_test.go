package bngsocket

import (
	"reflect"
	"testing"
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
	if isErrorType(reflect.TypeFor[error]()) {
		t.Fatal("isErrorType is broken, commitet value was error")
	}
	if isErrorType(reflect.TypeFor[int]()) {
		t.Fatal("isErrorType is broken, commitet value wasnt error")
	}
}

func testIsValidMessagePackType(t *testing.T) {
	// Es wird geprüft wie die Funktion auf Fehler reagiert
	t.Log("Test nill value without reflecting")
	err := isValidMessagePackType(nil)
	if err == nil {
		t.Log("Broken isValidMessagePackType function, accept nill values without reflecting")
	}
	if err.Error() != "passed Value was nil, not allowed" {
		t.Log("Broken isValidMessagePackType function, accept nill values without reflecting")
	}

	// Die Einfachen Zulässigen Datentypen werden gepürft
	t.Log("Test simple datatypes")
	if err := isValidMessagePackType(reflect.TypeFor[int]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[int8]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[int16]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[int32]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[int64]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[uint]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[uint8]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[uint16]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[uint32]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[uint64]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[float32]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[float64]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[bool]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[string]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[byte]()); err != nil {
		t.Fatal(err)
	}
	/*
		if err := isValidMessagePackType(reflect.TypeFor[TestObject]()); err != nil {
			t.Fatal(err)
		}
	*/
	if err := isValidMessagePackType(reflect.TypeFor[error]()); err != nil {
		t.Fatal(err)
	}

	// Die Zulässigen Maps werden gepürft
	t.Log("Test maps")
	if err := isValidMessagePackType(reflect.TypeFor[map[int]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[int8]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[int16]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[int32]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[int64]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[uint]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[uint16]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[uint32]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[uint64]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[float32]interface{}]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[map[string]interface{}]()); err != nil {
		t.Fatal(err)
	}
	/*
		if err := isValidMessagePackType(reflect.TypeFor[map[string]TestObject]()); err != nil {
			t.Fatal(err)
		}
	*/

	// Die Zulässigen Slices werden geprüft
	t.Log("Test slices")
	if err := isValidMessagePackType(reflect.TypeFor[[]int]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]int8]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]int16]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]int32]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]int64]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]uint]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]uint8]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]uint16]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]uint32]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]uint64]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]float32]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]float64]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]bool]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]string]()); err != nil {
		t.Fatal(err)
	}
	if err := isValidMessagePackType(reflect.TypeFor[[]byte]()); err != nil {
		t.Fatal(err)
	}
	/*
		if err := isValidMessagePackType(reflect.TypeFor[[]TestObject]()); err != nil {
			t.Fatal(err)
		}
	*/
	if err := isValidMessagePackType(reflect.TypeFor[[]error]()); err != nil {
		t.Fatal(err)
	}
}

func testValidateRPCFunction(t *testing.T) {
	// Es wird eine Zulässige Testfunktion auf der Registrierer Seite
	correctFunctionOnRegisterSide := func(req *BngRequest) error {
		return nil
	}
	value := reflect.ValueOf(correctFunctionOnRegisterSide)
	typeof := value.Type()
	if err := validateRPCFunction(value, typeof, true); err != nil {
		t.Fatal(err.Error())
	}

	// Es wird eine nicht Zulässige Testfunktion auf der Register Seite registriert
	incorrectFunctionOnRegisterSide := func(req *BngRequest) int {
		return -1
	}
	value = reflect.ValueOf(incorrectFunctionOnRegisterSide)
	typeof = value.Type()
	if err := validateRPCFunction(value, typeof, true); err == nil {
		t.Fatal("Invalid function testing failed, function was allowed from fucntion, function ist not correct!")
	}
}

func TestUtils(t *testing.T) {
	// Die Funktion "isErrorType" wird getestet
	t.Log("Test 'isErrorType' function")
	testIsErrorType(t)

	// Die Funktion "isValidMessagePackType" wird getestet
	t.Log("Test 'isValidMessagePackType' function")
	testIsValidMessagePackType(t)

	// Die Funktion "validateRPCFunction" wird getestet
	t.Log("Test 'validateRPCFunction' function")
	testValidateRPCFunction(t)
}
