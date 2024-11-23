package sockettests

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/custodia-cenv/bngsocket-go"
)

var (
	serverWait = new(sync.WaitGroup)
	clientWait = new(sync.WaitGroup)
	mainWait   = new(sync.WaitGroup)
)

func serveChannelConnection_ClientSideRPC(conn *bngsocket.BngConn) {
	_, err := conn.CallFunction("test", []interface{}{100000}, []reflect.Type{})
	if err != nil {
		panic(err)
	}

	exampleMap := map[string]int{
		"Apples":  5,
		"Bananas": 12,
		"Oranges": 7,
	}

	_, err = conn.CallFunction("test2", []interface{}{100, []string{"test wert"}, exampleMap}, []reflect.Type{})
	if err != nil {
		panic(err)
	}

	clientWait.Done()
}

func serveConn_ClientSideRPC(conn net.Conn) {
	// Die Verbindung wird geupgradet
	fmt.Println("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	serverWait.Wait()

	go func() {
		fmt.Println("Client: Moinitoring gestartet")
		err := bngsocket.MonitorConnection(upgrConn)
		if err != nil {
			if err != bngsocket.ErrConnectionClosedEOF {
				fmt.Println("Client: Moinitoring ist fehlgeschlagen:" + err.Error())
			}
		}
		fmt.Println("Client: Moinitoring geschlossen")
		mainWait.Done()
	}()

	go serveChannelConnection_ClientSideRPC(upgrConn)
}

func serveConn_ServerSideRPC(conn net.Conn) {
	// Die Verbindung wird geupgradet
	fmt.Println("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Es wird eine Routine gestartet, diese Routine Signalisiert
	go func() {
		fmt.Println("Server: Moinitoring gestartet")
		err := bngsocket.MonitorConnection(upgrConn)
		if err != nil {
			if err != bngsocket.ErrConnectionClosedEOF {
				fmt.Println("Server: Moinitoring ist fehlgeschlagen:" + err.Error())
			}
		}
		fmt.Println("Server: Moinitoring geschlossen")
		//mainWait.Done()
	}()

	// Es wird eine neue RPC Funktion Registriert
	err = upgrConn.RegisterFunction("test", func(req *bngsocket.BngRequest, version int) error {
		fmt.Println("CALL TEST")
		return nil
	})
	if err != nil {
		panic(err)
	}

	err = upgrConn.RegisterFunction("test2", func(req *bngsocket.BngRequest, a int, strslice []string, exampleMap map[string]int) error {
		fmt.Println(a, strslice, exampleMap)
		return nil
	})
	if err != nil {
		panic(err)
	}

	serverWait.Done()
	clientWait.Wait()
	upgrConn.Close()
}

// TestRPCSocket verwendet Unix-Sockets anstelle von net.Pipe()
func TestRPCSocket(t *testing.T) {
	bngsocket.DebugSetPrintFunction(func(v ...any) {
		fmt.Println(v...)
	})

	// Erstellen eines temporären Unix-Socket-Pfads
	socketDir := os.TempDir()
	socketPath := filepath.Join(socketDir, fmt.Sprintf("test_socket_%d.sock", time.Now().UnixNano()))

	// Sicherstellen, dass der Socket-Pfad nach dem Test entfernt wird
	defer os.Remove(socketPath)

	// Erstellen eines Unix-Socket-Listeners
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Fehler beim Erstellen des Unix-Socket-Listeners: %v", err)
	}
	defer listener.Close()

	mainWait.Add(2)
	clientWait.Add(1)
	serverWait.Add(1)

	// Starten des Server-Seitigen RPC in einer separaten Goroutine
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Errorf("Fehler beim Akzeptieren der Verbindung: %v", err)
			return
		}
		defer conn.Close()
		serveConn_ServerSideRPC(conn)
		mainWait.Done()
	}()

	// Warten, bis der Server bereit ist
	time.Sleep(100 * time.Millisecond)

	// Herstellen der Client-Verbindung zum Unix-Socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Fehler beim Verbinden zum Unix-Socket: %v", err)
	}
	defer conn.Close()

	// Starten des Client-Seitigen RPC
	go func() {
		serveConn_ClientSideRPC(conn)
		mainWait.Done()
	}()

	// Warten auf den Abschluss aller Goroutinen
	mainWait.Wait()
}
