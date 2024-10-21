package tests

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/CustodiaJS/bngsocket"
)

func acceptNewChannelRequests(channel *bngsocket.BngConnChannelListener) {
	// Diese Schleife nimmt eintreffende Datensätze entgegegen

}

func serveConnServerChannel(conn net.Conn) {
	// Die Verbindung wird geupgradet
	bngsocket.DebugPrint("Verbindung upgraden")
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println("Fehler beim Upgraden: " + err.Error())
		return
	}

	// Es wird ein neuer Channel bereitgestellt
	testChannel, err := upgrConn.OpenChannelListener("test-channel")
	if err != nil {
		bngsocket.DebugPrint(err.Error())
		return
	}

	// Es wird auf neu Eintreffende Channel Anfragen geantwortet
	go acceptNewChannelRequests(testChannel)
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
