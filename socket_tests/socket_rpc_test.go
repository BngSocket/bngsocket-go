package sockettests

import (
	"errors"
	"fmt"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/CustodiaJS/bngsocket"
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

	_, terr := conn.CallFunction("testNoneExisting", []interface{}{100000}, []reflect.Type{})
	if terr != nil {
		if !errors.Is(terr, bngsocket.ErrUnkownRpcFunction) {
			panic(terr.Error() == bngsocket.ErrUnkownRpcFunction.Error())
		}
	}

	testfunc := func(req *bngsocket.BngRequest) error {
		fmt.Println("PROXY_CALL")
		return nil
	}

	_, err = conn.CallFunction("test2", []interface{}{testfunc}, []reflect.Type{})
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
		mainWait.Done()
	}()

	// Es wird eine neue RPC Funktion Registriert
	err = upgrConn.RegisterFunction("test", func(req *bngsocket.BngRequest, version int) error {
		fmt.Println("CALL TEST")
		return nil
	})
	if err != nil {
		panic(err)
	}

	err = upgrConn.RegisterFunction("test2", func(req *bngsocket.BngRequest, testfunc func() error) error {
		testfunc()
		return nil
	})
	if err != nil {
		panic(err)
	}

	serverWait.Done()
	clientWait.Wait()
	upgrConn.Close()
}

func TestRPCSocket(t *testing.T) {
	bngsocket.DebugSetPrintFunction(t.Log)

	conn1, conn2 := net.Pipe()

	mainWait.Add(2)
	clientWait.Add(1)
	serverWait.Add(1)
	go serveConn_ServerSideRPC(conn1)
	time.Sleep(100 * time.Millisecond)
	serveConn_ClientSideRPC(conn2)
	mainWait.Wait()
}
