package tests

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/CustodiaJS/bngsocket"
)

func serveChannelConnection(channel *bngsocket.BngConnChannel) {
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

}

func serveConnClientChannel(conn net.Conn) {
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

	for {
		// Es wird auf neue Verbindungen gewartet
		channel, err := listener.Accept()
		if err != nil {
			bngsocket.DebugPrint(err.Error())
			os.Exit(1)
		}

		// Es wird auf Eintreffende Daten gewartet
		go serveChannelConnection(channel)
	}
}

func TestChannelClient(t *testing.T) {
	bngsocket.DebugSetPrintFunction(t.Log)

	// Der Unix Socket wird geöffnet
	bngsocket.DebugPrint("Connect to Unix Socket: /tmp/unixsock")
	conn, err := net.Dial("unix", "/tmp/unixsock")
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	serveConnClientChannel(conn)
}
