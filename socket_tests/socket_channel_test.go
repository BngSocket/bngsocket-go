package sockettests

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
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Die Eingetroffenen Daten werden ausgelesen
	data := readed[:n]
	fmt.Println(data)
	wgt.Done()
}

func serveConn_ClientSide(conn net.Conn) {
	// Die Verbindung wird geupgradet
	fmt.Println("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Es wird ein neuer Channel erzeugt
	listener, err := upgrConn.OpenChannelListener("test-channel")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Es wird auf neue Verbindungen gewartet
	channel, err := listener.Accept()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Es wird auf Eintreffende Daten gewartet
	wgt := new(sync.WaitGroup)
	wgt.Add(2)

	go func() {
		fmt.Println("Client: Moinitoring gestartet")
		err := bngsocket.MonitorConnection(upgrConn)
		if err != nil {
			if err != bngsocket.ErrConnectionClosedEOF {
				fmt.Println("Client: Moinitoring ist fehlgeschlagen:" + err.Error())
			}
		}
		fmt.Println("Client: Moinitoring geschlossen")
		wgt.Done()
	}()

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
		wg.Done()
	}()

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
	upgrConn.Close()
}

func TestChannelSocket(t *testing.T) {
	bngsocket.DebugSetPrintFunction(t.Log)

	conn1, conn2 := net.Pipe()

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go serveConn_ServerSide(conn1, wg)
	time.Sleep(100 * time.Millisecond)
	serveConn_ClientSide(conn2)
	wg.Wait()
}
