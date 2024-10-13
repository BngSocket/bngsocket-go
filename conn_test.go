package bngsocket

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
)

const SOCKET_PATH = "/tmp/testunixsocket"

func client_channel_test(t *testing.T, upgrConn *BngConn) error {
	// Es wird versucht auf einen Channel zu Joinen
	channel, err := upgrConn.JoinChannel("test-channel")
	if err != nil {
		return err
	}
	t.Log("Client:", channel.GetSessionId())

	// Es werden Daten in diesem Channel übertragen
	b := []byte("hallo welt")
	n, err := channel.Write(b)
	if err != nil {
		return err
	}
	t.Log("Write complete:", n, "of", len(b))

	// Der Gegenseite wird geantwortet
	data := make([]byte, 4096)
	i, err := channel.Read(data)
	if err != nil {
		return err
	}
	if !bytes.Equal(data[:i], b) {
		return fmt.Errorf("invald data resend")
	}

	// Der Channel wird geschlossen
	if closeErr := channel.Close(); closeErr != nil {
		return closeErr
	}

	// Es ist kein Fehler aufgetreten
	return nil
}

func server_channel_test(t *testing.T, upgrConn *BngConn) error {
	// Es wird ein neuer Channel bereitgestellt
	testChannel, err := upgrConn.OpenChannelListener("test-channel")
	if err != nil {
		return err
	}

	// Es wird eine Routine gestartet welche neue Anfrage auf dem Channel entgegen nimmt
	go func() {
		// Es wird auf neue Channel Joins gewartet
		t.Log("Warte auf neue eingehende Channel anfrage")
		chann, err := testChannel.Accept()
		if err != nil {
			panic(err)
		}
		t.Log("Server:", chann.GetSessionId())

		// Es wird auf eintreffende Daten gewartet
		data := make([]byte, 4096)
		r, err := chann.Read(data)
		if err != nil {
			panic(err)
		}

		// Die Daten werden zurückgesendet
		if i, _ := chann.Write(data[:r]); i != len(data[:r]) {
			panic("rewrite failed")
		}
	}()

	return nil
}

func server_rpc_test(_ *testing.T, upgrConn *BngConn) error {
	// Es wird eine neue Funktion Registriert
	err := upgrConn.RegisterFunction("test-function", func(req *BngRequest) error {
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func client_rpc_test(_ *testing.T, upgrConn *BngConn) error {
	// Es wird eine neue Funktion Registriert
	err := upgrConn.RegisterFunction("test-function", func(req *BngRequest) error {
		return nil
	})
	if err != nil {
		return err
	}
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
	t.Log("Server: Auf neue Verbindung warten")
	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Fehler beim Akzeptieren der Verbindung:", err)
		wg.Done()
		return
	}

	// Die Verbindung wird geupgradet
	t.Log("Server: Verbindung upgraden")
	upgrConn, err := UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println("Fehler beim Upgraden: " + err.Error())
		wg.Done()
		return
	}

	// Der Channel Test wird durchgeführt
	t.Log("Server: Channel testing")
	if err := server_channel_test(t, upgrConn); err != nil {
		fmt.Println(err)
		wg.Done()
	}

	// Die RPC Funktionen werden gestet
	t.Log("Server: RPC testing")
	if err := server_rpc_test(t, upgrConn); err != nil {
		panic(err)
	}

	wg.Done()
}

// Tests die Clientseite
func Client(t *testing.T, wg *sync.WaitGroup) {
	// Es wird eine Verbindung mit dem Server hergestellt
	t.Log("Client: Versuche neue Verbindung herzustellen")
	conn, err := net.Dial("unix", SOCKET_PATH)
	if err != nil {
		fmt.Println("Fehler beim Verbinden zum Socket:", err)
		wg.Done()
		return
	}
	defer conn.Close()

	// Die Verbindung wird geupgradet und die Channel Tests werden durchgeführt
	t.Log("Client: Verbindung upgraden")
	upgrConn, err := UpgradeSocketToBngConn(conn)
	if err != nil {
		fmt.Println("Fehler beim Upgraden: " + err.Error())
		wg.Done()
		return
	}

	// Der Channel Test wird durchgeführt
	t.Log("Client: Channel test")
	if err := client_channel_test(t, upgrConn); err != nil {
		fmt.Println(err)
		wg.Done()
	}

	// Die RPC Funktionen werden gestet
	t.Log("Client: RPC testing")
	if err := client_rpc_test(t, upgrConn); err != nil {
		panic(err)
	}

	wg.Done()
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
