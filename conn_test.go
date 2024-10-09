package bngsocket

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
)

const SOCKET_PATH = "/tmp/testunixsocket"

func client_channel_test(_ *testing.T, upgrConn *BngConn) error {
	// Es wird versucht auf einen Channel zu Joinen
	channel, err := upgrConn.JoinChannel("test-channel")
	if err != nil {
		return err
	}
	fmt.Println("Client:", channel.GetSessionId())

	// Es werden Daten in diesem Channel übertragen
	b := []byte("hallo welt")
	n, err := channel.Write(b)
	if err != nil {
		return err
	}
	fmt.Println("Write complete:", n, "of", len(b))

	// Der Channel wird geschlossen
	if closeErr := channel.Close(); closeErr != nil {
		return closeErr
	}

	// Es ist kein Fehler aufgetreten
	return nil
}

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
	upgrConn, err := UpgradeSocketToBngConn(conn)
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

	// Die Verbindung wird geupgradet und die Channel Tests werden durchgeführt
	upgrConn, err := UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println("Fehler beim Upgraden: " + err.Error())
		wg.Done()
		return
	}
	if err := client_channel_test(t, upgrConn); err != nil {
		fmt.Println(err)
		wg.Done()
	}
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
