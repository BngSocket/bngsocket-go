package bngsocket

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/CustodiaJS/bngsocket"
)

const SOCKET_PATH = "/tmp/testunixsocket"

func client_channel_test(t *testing.T, upgrConn *bngsocket.BngConn) error {
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

func server_channel_test(t *testing.T, upgrConn *bngsocket.BngConn) error {
	// Es wird ein neuer Channel bereitgestellt
	testChannel, err := upgrConn.OpenChannelListener("test-channel")
	if err != nil {
		return err
	}

	wgt := new(sync.WaitGroup)
	wgt.Add(1)

	// Es wird eine Routine gestartet welche neue Anfrage auf dem Channel entgegen nimmt
	go func() {
		defer wgt.Done()

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
		if i, err := chann.Write(data[:r]); i != len(data[:r]) {
			panic("rewrite failed: " + err.Error())
		}
	}()

	wgt.Wait()

	return nil
}

func server_rpc_test(_ *testing.T, upgrConn *bngsocket.BngConn, server_channel_wait *sync.WaitGroup) error {
	// Es wird eine Funktion Registriert welchen von jedem Godatentypen einen Rückgabwert zurückgibt
	err := upgrConn.RegisterFunction("test-function-with-returns", func(req *bngsocket.BngRequest, i int, u uint, f float64, str string, b bool, bytes []byte, mapv map[string]interface{}) (int, uint, float64, string, bool, []byte, map[string]interface{}, error) {
		fmt.Println("RPC Function Body:", i, u, f, str, b, string(bytes), mapv)
		return i, u, f, str, b, bytes, mapv, nil
	})
	if err != nil {
		return err
	}

	server_channel_wait.Done()
	return nil
}

func client_rpc_test(_ *testing.T, upgrConn *bngsocket.BngConn) error {
	// Die RPC Funktion mit den Standard Go Datentypen wird aufgerufen
	returnDataTypes := make([]reflect.Type, 0)
	data := map[string]interface{}{"name": "Max Mustermann", "age": 29, "isAdmin": true, "scores": []int{90, 85, 92}}
	returnDataTypes = append(returnDataTypes, reflect.TypeFor[int](), reflect.TypeFor[uint](), reflect.TypeFor[float64](), reflect.TypeFor[string](), reflect.TypeFor[bool](), reflect.TypeFor[[]byte](), reflect.TypeFor[map[string]interface{}]())
	datax, err := upgrConn.CallFunction("test-function-with-returns", append(make([]interface{}, 0), 100, 100, 1.2, "hallo welt str", true, []byte("hallo welt bytes"), data), returnDataTypes)
	if err != nil {
		return err
	}
	for _, item := range datax {
		fmt.Println("Return:", item)
	}

	// Es ist kein fehler aufgetreten
	return nil
}

// Testet die Serverseite
func Server(t *testing.T, wg *sync.WaitGroup, swg *sync.WaitGroup, server_channel_wait *sync.WaitGroup) {
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
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
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
	if err := server_rpc_test(t, upgrConn, server_channel_wait); err != nil {
		panic(err)
	}

	wg.Done()
}

// Tests die Clientseite
func Client(t *testing.T, wg *sync.WaitGroup, server_channel_wait *sync.WaitGroup) {
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
	upgrConn, err := bngsocket.UpgradeSocketToBngConn(conn)
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
	server_channel_wait.Wait()

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
	server_channel_wait := &sync.WaitGroup{}
	server_channel_wait.Add(1)
	swg.Add(1)
	wg.Add(2)
	go Server(t, wg, swg, server_channel_wait)
	swg.Wait()
	go Client(t, wg, server_channel_wait)
	wg.Wait()
}
