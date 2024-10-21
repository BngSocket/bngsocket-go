package tests

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"github.com/CustodiaJS/bngsocket"
)

func rpcClient(_ *testing.T, upgrConn *bngsocket.BngConn) error {
	// Die RPC Funktion mit den Standard Go Datentypen wird aufgerufen
	returnDataTypes := make([]reflect.Type, 0)
	data := map[string]interface{}{"name": "Max Mustermann", "age": 29, "isAdmin": true, "scores": []int{90, 85, 92}}
	returnDataTypes = append(returnDataTypes, reflect.TypeFor[int](), reflect.TypeFor[uint](), reflect.TypeFor[float64](), reflect.TypeFor[string](), reflect.TypeFor[bool](), reflect.TypeFor[[]byte](), reflect.TypeFor[map[string]interface{}]())
	datax, err := upgrConn.CallFunction("test-function-with-returns", append(make([]interface{}, 0), 100, 100, 1.2, "hallo welt str", true, []byte("hallo welt bytes"), data), returnDataTypes)
	if err != nil {
		return err
	}
	for _, item := range datax {
		fmt.Println("Return:", item)
	}

	// Es ist kein fehler aufgetreten
	return nil
}

// Tests die Clientseite
func TestClient(t *testing.T) {
	bngsocket.DebugSetPrintFunction(t.Log)
	
	// Der Unix Socket wird geöffnet
	conn, err := net.Dial("unix", "/tmp/unixsock")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// Die Verbindung wird geupgradet und die Channel Tests werden durchgeführt
	t.Log("Client: Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println("Fehler beim Upgraden: " + err.Error())
		return
	}

	// Die RPC Funktionen werden gestet
	t.Log("Client: RPC testing")
	if err := rpcClient(t, upgrConn); err != nil {
		panic(err)
	}
}
