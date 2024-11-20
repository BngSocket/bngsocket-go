package bngsocket

import (
	"bufio"
	"fmt"

	"github.com/CustodiaJS/bngsocket/transport"
	"github.com/vmihailenco/msgpack/v5"
)

func writePacketACK(o *BngConn) error {
	writer := bufio.NewWriter(o.conn)
	ack := []byte("ACK")

	// Schreibe das ACK
	if _, err := writer.Write(ack); err != nil {
		return fmt.Errorf("%w: %v", ErrWriteACK, err)
	}

	// Flushe den Writer
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("%w: %v", ErrFlushACK, err)
	}

	_DebugPrint(fmt.Sprintf("BngConn(%s): ACK sent", o._innerhid))
	return nil
}

// convertAndWriteBytesIntoChan wandelt einen Go Datensatz in Transportable Bytes um
func convertAndWriteBytesIntoChan(conn *BngConn, data interface{}) error {
	// Den RpcRequest in Bytes serialisieren.
	bdata, err := msgpack.Marshal(data)
	if err != nil {
		return fmt.Errorf("channelWriteACK[0]: %s", err.Error())
	}

	// Die Bytes in den Schreibkanal des Sockets schreiben.
	if err := writeBytesIntoSocketConn(conn, bdata); err != nil {
		return fmt.Errorf("channelWriteACK[1]: %s", err.Error())
	}

	return nil
}

// responseUnkownChannel sendet eine Antwort zurück, wenn ein unbekannter Kanal angefordert wurde.
func responseUnkownChannel(conn *BngConn, sourceId string) error {
	rt := &transport.ChannelRequestResponse{
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
	rt := &transport.ChannelRequestResponse{
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
	rt := &transport.ChannlSessionTransportSignal{
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
	rt := &transport.ChannelSessionDataTransport{
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
	if err := writeBytesIntoSocketConn(socket, bdata); err != nil {
		return 0, -1, fmt.Errorf("channelDataTransport[1]: %s", err.Error())
	}

	// Die Paket-ID und die Größe der gesendeten Daten zurückgeben.
	return rt.PackageId, len(data), nil
}

// channelWriteACK sendet ein ACK (Acknowledgment) für ein bestimmtes Paket über die BNG-Verbindung.
func channelWriteACK(conn *BngConn, pid uint64, sessionId string) error {
	rt := &transport.ChannelTransportStateResponse{
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

// socketWriteRpcSuccessResponse antwortet auf ein RPC Request mit einem Response
func socketWriteRpcSuccessResponse(conn *BngConn, value []*transport.RpcDataCapsle, id string) error {
	rt := &transport.RpcResponse{
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

// socketWriteRpcErrorResponse sendet ein Fehler auf einen RPC Request
func socketWriteRpcErrorResponse(conn *BngConn, errstr string, id string) error {
	rt := &transport.RpcResponse{
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

// channelWriteCloseSignal schreibt ein Close Signal an die Gegenseite
func channelWriteCloseSignal(conn *BngConn, channelSessionId string) error {
	rt := &transport.ChannlSessionTransportSignal{
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

// channelWriteACKForJoin bestätigt den Join auf einen Channel
func channelWriteACKForJoin(conn *BngConn, channelSessionId string) error {
	rt := &transport.ChannlSessionTransportSignal{
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
