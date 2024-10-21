package tests

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/CustodiaJS/bngsocket"
)

// Testet die Serverseite
func rpcServer(conn1 net.Conn) {
	// Die Verbindung wird geupgradet
	bngsocket.DebugPrint("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn1)
	if err != nil {
		fmt.Println("Fehler beim Upgraden: " + err.Error())
		return
	}

	// Es wird eine Funktion Registriert welchen von jedem Godatentypen einen Rückgabwert zurückgibt
	calledFunction := new(sync.WaitGroup)
	calledFunction.Add(1)
	err = upgrConn.RegisterFunction("test-function-with-returns", func(req *bngsocket.BngRequest, i int, u uint, f float64, str string, b bool, bytes []byte, mapv map[string]interface{}) (int, uint, float64, string, bool, []byte, map[string]interface{}, error) {
		defer func() {
			go func() {
				time.Sleep(1 * time.Second)
				calledFunction.Done()
			}()
		}()
		return i, u, f, str, b, bytes, mapv, nil
	})
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		return
	}
	calledFunction.Wait()
}

func TestServer(t *testing.T) {
	bngsocket.DebugSetPrintFunction(t.Log)
	t.Log("Open UnixSocket")
	os.Remove("/tmp/unixsock")
	l, err := net.ListenUnix("unix", &net.UnixAddr{Name: "/tmp/unixsock", Net: "unix"})
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		return
	}
	defer l.Close()

	for {
		// Anfrage wird angenommen
		t.Log("Wait of new conenction")
		conn, err := l.Accept()
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		// Der RPC Server wird gestartet
		go rpcServer(conn)
	}
}
