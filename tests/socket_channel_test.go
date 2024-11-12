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

func serveChannelConnection_ClientSide(channel *bngsocket.BngConnChannel, wgt *sync.WaitGroup) {
	// Es wird auf die Eintreffenden Daten gewartet
	readed := make([]byte, 1024)
	n, err := channel.Read(readed)
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		os.Exit(1)
	}

	// Die Eingetroffenen Daten werden ausgelesen
	data := readed[:n]
	fmt.Println(data)
	wgt.Done()
}

func serveConn_ClientSide(conn net.Conn) {
	// Die Verbindung wird geupgradet
	bngsocket.DebugPrint("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		os.Exit(1)
	}

	// Es wird ein neuer Channel erzeugt
	listener, err := upgrConn.OpenChannelListener("test-channel")
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		os.Exit(1)
	}

	// Es wird auf neue Verbindungen gewartet
	channel, err := listener.Accept()
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		os.Exit(1)
	}

	// Es wird auf Eintreffende Daten gewartet
	wgt := new(sync.WaitGroup)
	wgt.Add(1)
	go serveChannelConnection_ClientSide(channel, wgt)
	wgt.Wait()
}

func serveConn_ServerSide(conn net.Conn) {
	// Die Verbindung wird geupgradet
	bngsocket.DebugPrint("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		return
	}

	// Es wird eine ausgehende Verbindung hergestellt
	channel, err := upgrConn.JoinChannel("test-channel")
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		return
	}

	// Es wird ein HalloWelt Paket an den Client gesendet
	_, err = channel.Write([]byte("HalloWelt"))
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		return
	}
}

func TestChannelSocket(t *testing.T) {
	bngsocket.DebugSetPrintFunction(t.Log)

	conn1, conn2 := net.Pipe()

	go serveConn_ServerSide(conn1)
	time.Sleep(100 * time.Millisecond)
	serveConn_ClientSide(conn2)
}
