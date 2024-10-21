package tests

import (
	"bytes"
	"fmt"
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
