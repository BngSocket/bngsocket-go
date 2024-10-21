package bngsocket

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

// writeBytesIntoChan schreibt ein Byte-Array in den Schreibkanal des angegebenen Sockets.
// Es gibt einen Fehler zurück, wenn der Socket oder der Schreibkanal nicht verfügbar ist.
func writeBytesIntoChan(socket *BngConn, data []byte) error {
	// Überprüfen, ob der Socket vorhanden ist.
	if socket == nil {
		return fmt.Errorf("socket ist null, not allowed")
	}

	// Überprüfen, ob der Schreibkanal vorhanden ist.
	if socket.writingChan == nil {
		return fmt.Errorf("closed write chan")
	}

	// Einen Kanal für den Schreibstatus erstellen.
	resolveChan := make(chan *writingState, 1)
	preparedData := &dataWritingResolver{data: data, waitOfResolve: resolveChan}

	// Die Daten in den Schreibkanal senden.
	if ok := socket.writingChan.Enter(preparedData); !ok {
		return fmt.Errorf("writeBytesIntoChan: error by sending data through channel %v", socket.writingChan.IsOpen())
	}

	// Auf den Status der Sendung warten.
	resolve := <-resolveChan

	// Überprüfen, ob ein Fehler während des Schreibens aufgetreten ist.
	if resolve.err != nil {
		return fmt.Errorf("writeBytesIntoChan: %s", resolve.err.Error())
	}

	// Den Kanal schließen, da die Sendung abgeschlossen ist.
	close(resolveChan)

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
	return nil
}

func convertAndWriteBytesIntoChan(conn *BngConn, data interface{}) error {
	// Den RpcRequest in Bytes serialisieren.
	bdata, err := msgpack.Marshal(data)
	if err != nil {
		return fmt.Errorf("channelWriteACK[0]: %s", err.Error())
	}

	// Die Bytes in den Schreibkanal des Sockets schreiben.
	if err := writeBytesIntoChan(conn, bdata); err != nil {
		return fmt.Errorf("channelWriteACK[1]: %s", err.Error())
	}

	return nil
}

// responseUnkownChannel sendet eine Antwort zurück, wenn ein unbekannter Kanal angefordert wurde.
func responseUnkownChannel(conn *BngConn, sourceId string) error {
	rt := &ChannelRequestResponse{
		Type:                "chreqresp",       // Typ der Antwort
		ReqId:               sourceId,          // ID der Anfrage
		NotAcceptedByReason: "#unkown_channel", // Grund für die Ablehnung
	}

	err := convertAndWriteBytesIntoChan(conn, rt)
	if err != nil {
		return err
	}

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
	return nil
}

// responseNewChannelSession sendet eine Antwort zurück, wenn eine neue Channel-Sitzung registriert wird.
func responseNewChannelSession(conn *BngConn, channelRequestId string, channelSessionId string) error {
	rt := &ChannelRequestResponse{
		Type:      "chreqresp",      // Typ der Antwort
		ReqId:     channelRequestId, // ID der Anfrage
		ChannelId: channelSessionId, // ID der neuen Channel-Sitzung
	}

	err := convertAndWriteBytesIntoChan(conn, rt)
	if err != nil {
		return err
	}

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
	return nil
}

// responseChannelNotOpen sendet ein Signal zurück, dass der angegebene Channel nicht geöffnet ist.
func responseChannelNotOpen(conn *BngConn, channelId string) error {
	rt := &ChannlSessionTransportSignal{
		Type:             "chsig",   // Typ des Signals
		ChannelSessionId: channelId, // ID des nicht geöffneten Channels
		Signal:           0,         // Signalwert (0 bedeutet "nicht geöffnet")
	}

	err := convertAndWriteBytesIntoChan(conn, rt)
	if err != nil {
		return err
	}

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
	return nil
}

// channelDataTransport sendet Daten über einen bestimmten Channel und gibt die Paket-ID und die Größe der Daten zurück.
func channelDataTransport(socket *BngConn, data []byte, channelSessionId string) (uint64, int, error) {
	rt := &ChannelSessionDataTransport{
		Type:             "chst",           // Typ der Datenübertragung
		ChannelSessionId: channelSessionId, // ID der Channel-Sitzung
		PackageId:        0,                // Paket-ID (wird im weiteren Verlauf gesetzt)
		Body:             data,             // Die zu übertragenden Daten
	}

	// Die Daten in Bytes serialisieren.
	bdata, err := msgpack.Marshal(rt)
	if err != nil {
		return 0, -1, fmt.Errorf("channelDataTransport[0]: %s", err.Error())
	}

	// Die Bytes in den Schreibkanal des Sockets schreiben.
	if err := writeBytesIntoChan(socket, bdata); err != nil {
		return 0, -1, fmt.Errorf("channelDataTransport[1]: %s", err.Error())
	}

	// Die Paket-ID und die Größe der gesendeten Daten zurückgeben.
	return rt.PackageId, len(data), nil
}

// channelWriteACK sendet ein ACK (Acknowledgment) für ein bestimmtes Paket über die BNG-Verbindung.
func channelWriteACK(conn *BngConn, pid uint64, sessionId string) error {
	rt := &ChannelTransportStateResponse{
		Type:             "chtsr",   // Typ der ACK-Antwort
		ChannelSessionId: sessionId, // ID der Channel-Sitzung
		PackageId:        pid,       // ID des Pakets, für das das ACK gesendet wird
		State:            0,         // Zustand (0 bedeutet ACK)
	}

	err := convertAndWriteBytesIntoChan(conn, rt)
	if err != nil {
		return err
	}

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
	return nil
}

func socketWriteRpcSuccessResponse(conn *BngConn, value []*RpcDataCapsle, id string) error {
	rt := &RpcResponse{
		Type:   "rpcres",
		Id:     id,
		Return: value,
	}

	err := convertAndWriteBytesIntoChan(conn, rt)
	if err != nil {
		return err
	}

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
	return nil
}

func socketWriteRpcErrorResponse(conn *BngConn, errstr string, id string) error {
	rt := &RpcResponse{
		Type:  "rpcres",
		Id:    id,
		Error: errstr,
	}

	err := convertAndWriteBytesIntoChan(conn, rt)
	if err != nil {
		return err
	}

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
	return nil
}

func channelWriteCloseSignal(conn *BngConn, channelSessionId string) error {
	rt := &ChannlSessionTransportSignal{
		Type:             "chsig",
		ChannelSessionId: channelSessionId,
		Signal:           0,
	}

	err := convertAndWriteBytesIntoChan(conn, rt)
	if err != nil {
		return err
	}

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
	return nil
}

func channelWriteACKForJoin(conn *BngConn, channelSessionId string) error {
	rt := &ChannlSessionTransportSignal{
		Type:             "chsig",
		ChannelSessionId: channelSessionId,
		Signal:           1,
	}

	err := convertAndWriteBytesIntoChan(conn, rt)
	if err != nil {
		return err
	}

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
	return nil
}
