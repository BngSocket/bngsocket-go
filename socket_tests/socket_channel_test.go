package sockettests

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/CustodiaJS/bngsocket"
)

var endlwait = new(sync.WaitGroup)
var waitOfServerChannel = new(sync.WaitGroup)

func serveChannelConnection_ClientSide(channel *bngsocket.BngConnChannel, wgt *sync.WaitGroup) {
	// Es wird auf die Eintreffenden Daten gewartet
	readed := make([]byte, 1024)
	n, err := channel.Read(readed)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Die Eingetroffenen Daten werden ausgelesen
	data := readed[:n]
	fmt.Println(string(data))
	endlwait.Done()
	wgt.Done()
}

func serveConn_ClientSide(conn net.Conn) {
	// Die Verbindung wird geupgradet
	fmt.Println("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println("T", err.Error())
		os.Exit(1)
	}

	// Es wird ein neuer Channel erzeugt
	listener, err := upgrConn.OpenChannelListener("test-channel")
	if err != nil {
		fmt.Println("R", err.Error())
		os.Exit(1)
	}

	waitOfServerChannel.Done()

	// Es wird auf neue Verbindungen gewartet
	channel, err := listener.Accept()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Es wird auf Eintreffende Daten gewartet
	wgt := new(sync.WaitGroup)
	wgt.Add(1)

	go serveChannelConnection_ClientSide(channel, wgt)
	wgt.Wait()
}

func serveConn_ServerSide(conn net.Conn, wg *sync.WaitGroup) {
	// Die Verbindung wird geupgradet
	fmt.Println("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	waitOfServerChannel.Wait()

	// Es wird eine ausgehende Verbindung hergestellt
	channel, err := upgrConn.JoinChannel("test-channel")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Es wird ein HalloWelt Paket an den Client gesendet
	_, err = channel.Write([]byte("HalloWelt"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	endlwait.Wait()
}

func TestChannelSocket(t *testing.T) {
	bngsocket.DebugSetPrintFunction(func(v ...any) {
		fmt.Println(v...)
	})

	for i := 0; i < 1; i++ {
		conn1, conn2 := net.Pipe()
		wg := new(sync.WaitGroup)
		endlwait.Add(1)
		wg.Add(1)
		waitOfServerChannel.Add(1)
		go serveConn_ServerSide(conn1, wg)
		serveConn_ClientSide(conn2)
		wg.Wait()
	}
}
