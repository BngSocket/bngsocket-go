package tests

import (
	"net"
	"testing"

	"github.com/CustodiaJS/bngsocket"
)

func serveConnClientChannel(conn net.Conn) {
	// Die Verbindung wird geupgradet
	bngsocket.DebugPrint("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		return
	}

	// Es wird ein neuer Channel erzeugt
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

	// Der RPC Server wird gestartet
	go serveConnClientChannel(conn)
}
