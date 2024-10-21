package tests

import (
	"bytes"
	"fmt"
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
	t.Log("Client: Write: hallo welt")
	b := []byte("hallo welt")
	n, err := channel.Write(b)
	if err != nil {
		return err
	}
	t.Log("Client: Write complete:", n, "of", len(b))

	// Der Gegenseite wird geantwortet
	t.Log("Write back")
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
		t.Log("Server: Reading")
		data := make([]byte, 4096)
		r, err := chann.Read(data)
		if err != nil {
			t.Error("Server: " + err.Error())
			t.Fail()
		}
		t.Log("Server: Reading done")

		// Die Daten werden zurückgesendet
		if i, err := chann.Write(data[:r]); i != len(data[:r]) {
			panic("rewrite failed: " + err.Error())
		}
	}()

	wgt.Wait()

	return nil
}
