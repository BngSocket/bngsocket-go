package bngsocket

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/CustodiaJS/bngsocket/transport"
	"github.com/vmihailenco/msgpack/v5"
)

// Wird verwendet um zu bestätigen dass ein Packet erfolgreich übertragen wurde
func writePackConfirmationACK(o *BngConn) error {
	// Byte senden
	_, err := conn.Write([]byte{ackType})
	if err != nil {
		return fmt.Errorf("error sending ACK/NACK: %w", err)
	}

	_DebugPrint(fmt.Sprintf("Sent ACK/NACK: %d", ackType))
	return nil
}

// writeBytesIntoSocketConn schreibt ein Byte-Array in den Schreibkanal des angegebenen Sockets.
// Es gibt einen Fehler zurück, wenn der Socket oder der Schreibkanal nicht verfügbar ist.
func writeBytesIntoSocketConn(o *BngConn, data []byte) error {
	// Überprüfen, ob der Socket vorhanden ist.
	if o == nil {
		return fmt.Errorf("socket ist null, not allowed")
	}

	// Daten in Chunks aufteilen
	chunks := splitDataIntoChunks(data, 4096)

	// Es wird auf den Mutex gewartet
	o.writerMutex.Lock()
	defer o.writerMutex.Unlock()

	// Es wird geprüft ob die Verbindung getrennt wurde
	if connectionIsClosed(o) {
		return io.EOF
	}

	// Die Chunks werden übertragen
	for _, chunk := range chunks {
		// Es wird geprüft ob die Verbindung getrennt wurde
		if connectionIsClosed(o) {
			return io.EOF
		}

		// Nachrichtentyp 'C' senden
		err := o.writer.WriteByte('C')
		if err != nil {
			// Der Fehler wird verarbeitet
			writeProcessErrorHandling(o, err)
			return err
		}

		// Chunk-Länge senden (2 Bytes)
		length := uint16(len(chunk))
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, length)
		_, err = o.writer.Write(lengthBytes)
		if err != nil {
			// Der Fehler wird verarbeitet
			writeProcessErrorHandling(o, err)
			return err
		}

		// Chunk-Daten senden
		_, err = o.writer.Write(chunk)
		if err != nil {
			// Der Fehler wird verarbeitet
			writeProcessErrorHandling(o, err)
			return err
		}

		// Flush, um sicherzustellen, dass die Daten gesendet werden
		err = o.writer.Flush()
		if err != nil {
			// Der Fehler wird verarbeitet
			writeProcessErrorHandling(o, err)
			return err
		}
	}

	// Es wird geprüft ob die Verbindung getrennt wurde
	if connectionIsClosed(o) {
		return io.EOF
	}

	// Nachrichtentyp 'L' senden, um das Ende der Nachricht zu signalisieren
	err := o.writer.WriteByte('L')
	if err != nil {
		// Der Fehler wird verarbeitet
		writeProcessErrorHandling(o, err)
		return err
	}

	// Die Übertragung wird fertigestellt
	err = o.writer.Flush()
	if err != nil {
		// Der Fehler wird verarbeitet
		writeProcessErrorHandling(o, err)
		return err
	}

	// Debug
	_DebugPrint(fmt.Sprintf("BngConn(%s): %d bytes writed", o._innerhid, len(data)))

	// Es ist kein Fehler aufgetreten, Rückgabe nil.
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
