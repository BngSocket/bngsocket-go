package bngsocket

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
)

const SOCKET_PATH = "/tmp/testunixsocket"

// Testet die Serverseite
func Server(t *testing.T, wg *sync.WaitGroup, swg *sync.WaitGroup) {
	if err := os.Remove(SOCKET_PATH); err != nil {
		fmt.Println(err)
	}

	// Erstelle einen neuen UNIX-Socket
	listener, err := net.Listen("unix", SOCKET_PATH)
	if err != nil {
		fmt.Println("Fehler beim Erstellen des Socket:", err)
		return
	}
	defer listener.Close()
	swg.Done()

	// Warte auf eine Verbindung
	fmt.Println("Auf neue Verbindung warten")
	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Fehler beim Akzeptieren der Verbindung:", err)
		wg.Done()
		return
	}

	// Die Verbindung wird geupgradet
	upgrConn, err := UpgradeSocketToBngSocket(conn)
	if err != nil {
		fmt.Println("Fehler beim Upgraden: " + err.Error())
		wg.Done()
		return
	}

	// Es wird ein neuer Channel bereitgestellt
	testChannel, err := upgrConn.OpenChannelListener("test-channel")
	if err != nil {
		fmt.Println("Fehler beim Channel Listener", err)
		wg.Done()
		return
	}

	// Es wird eine Routine gestartet welche neue Anfrage auf dem Channel entgegen nimmt
	go func() {
		// Es wird auf neue Channel Joins gewartet
		fmt.Println("Warte auf neue eingehende Channel anfrage")
		chann, err := testChannel.Accept()
		if err != nil {
			panic(err)
		}
		fmt.Println("Server:", chann.GetSessionId())
		data := make([]byte, 4096)
		r, err := chann.Read(data)
		if err != nil {
			panic(err)
		}
		fmt.Println(r, string(data[:r]))
	}()

	// Es wird eine neue Funktion Registriert
	err = upgrConn.RegisterFunction("test-function", func(req *BngRequest) error {
		return nil
	})
	if err != nil {
		fmt.Println("Fehler beim Registrieren der Einfachen Testfunktion")
		wg.Done()
		return
	}
}

// Tests die Clientseite
func Client(t *testing.T, wg *sync.WaitGroup) {
	// Es wird eine Verbindung mit dem Server hergestellt
	conn, err := net.Dial("unix", SOCKET_PATH)
	if err != nil {
		fmt.Println("Fehler beim Verbinden zum Socket:", err)
		wg.Done()
		return
	}
	defer conn.Close()

	// Die Verbindung wird geupgradet
	upgrConn, err := UpgradeSocketToBngSocket(conn)
	if err != nil {
		fmt.Println("Fehler beim Upgraden: " + err.Error())
		wg.Done()
		return
	}

	// Es wird ein neuer Channl Request gestartet
	channel, err := upgrConn.JoinChannel("test-channel")
	if err != nil {
		fmt.Println("Fehler beim Join auf den Test Channel: " + err.Error())
		wg.Done()
		return
	}
	fmt.Println("Client:", channel.GetSessionId())

	b := []byte("hallo welt")
	n, err := channel.Write(b)
	if err != nil {
		panic(err)
	}
	fmt.Println("Write complete:", n, "of", len(b))
}

// TestAdd prüft die Additionsfunktion.
func TestServer(t *testing.T) {
	swg := &sync.WaitGroup{}
	wg := &sync.WaitGroup{}
	swg.Add(1)
	wg.Add(2)
	go Server(t, wg, swg)
	swg.Wait()
	go Client(t, wg)
	wg.Wait()
}
