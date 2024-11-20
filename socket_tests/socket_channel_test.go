package sockettests

import (
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/CustodiaJS/bngsocket"
)

var (
	isServer = flag.Bool("server", false, "Starte den Test als Server")
)

func serveChannelConnection_ClientSide(channel *bngsocket.BngConnChannel) {
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
}

func serveConn_ClientSide(conn net.Conn) {
	// Die Verbindung wird geupgradet
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

	// Es wird auf neue Verbindungen gewartet
	channel, err := listener.Accept()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Es wird auf Eintreffende Daten gewartet
	wgt := new(sync.WaitGroup)
	wgt.Add(1)

	go serveChannelConnection_ClientSide(channel)
	wgt.Wait()
}

func serveConn_ServerSide(conn net.Conn) {
	// Die Verbindung wird geupgradet
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	time.Sleep(500 * time.Millisecond)

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
}

func TestChannelSocket(t *testing.T) {
	bngsocket.DebugSetPrintFunction(func(v ...any) {
		fmt.Println(v...)
	})

	// UNIX-Socket Pfad
	socketPath := "/tmp/test_bngsocket.sock"

	if *isServer {
		// Starte als Server
		os.Remove(socketPath)
		startServer(t, socketPath)
	} else {
		// Starte als Client
		startClient(t, socketPath)
	}
}

func startServer(t *testing.T, socketPath string) {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Fehler beim Erstellen des UNIX-Socket-Listeners: %v", err)
	}
	defer listener.Close()

	fmt.Println("Server gestartet, wartet auf Verbindungen:", socketPath)

	wg := new(sync.WaitGroup)
	wg.Add(1)

	for {
		conn, err := listener.Accept()
		if err != nil {
			t.Fatalf("fehler beim Akzeptieren der Verbindung: %v", err)
		}
		go serveConn_ServerSide(conn)
	}
}

func startClient(t *testing.T, socketPath string) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Fehler beim Verbinden zum UNIX-Socket: %v", err)
	}
	defer conn.Close()

	fmt.Println("Client gestartet, verbindet zum Server...")
	serveConn_ClientSide(conn)
	fmt.Println("Client beendet")
}
