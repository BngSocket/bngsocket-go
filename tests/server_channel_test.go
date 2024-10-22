package tests

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/CustodiaJS/bngsocket"
)

func serveConnServerChannel(conn net.Conn) {
	// Die Verbindung wird geupgradet
	bngsocket.DebugPrint("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println("Fehler beim Upgraden: " + err.Error())
		return
	}

	// Es wird versucht einen 
}

func TestChannelServer(t *testing.T) {
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
		go serveConnServerChannel(conn)
	}
}
