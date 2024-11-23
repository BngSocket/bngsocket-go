package bngsocket

import (
	"fmt"

	"github.com/custodia-cenv/bngsocket-go/transport"
	"github.com/vmihailenco/msgpack/v5"
)

// writePacketACK sendet ein ACK (Acknowledgment) über die Socket-Verbindung des BngConn-Objekts.
// Die Funktion schreibt das ACK-Signal in den Writer und flushed diesen, um sicherzustellen, dass die Daten gesendet werden.
// Nach dem Senden des ACKs wird eine Debug-Nachricht ausgegeben.
//
// Parameter:
//   - o *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Senden des ACKs ein Problem aufgetreten ist, ansonsten nil.
func writePacketACK(o *BngConn) error {
	o.writerMutex.Lock()
	defer o.writerMutex.Unlock()

	ack := []byte("ACK")

	// Schreibe das ACK
	if _, err := o.writer.Write(ack); err != nil {
		return fmt.Errorf("%w: %v", ErrWriteACK, err)
	}

	// Flushe den Writer
	if err := o.writer.Flush(); err != nil {
		return fmt.Errorf("%w: %v", ErrFlushACK, err)
	}

	_DebugPrint(fmt.Sprintf("BngConn(%s): ACK sent", o._innerhid))
	return nil
}

// convertAndWriteBytesIntoChan wandelt einen Go-Datensatz in transportierbare Bytes um und schreibt diese in den Schreibkanal des BngConn-Objekts.
// Die Funktion serialisiert die Daten mit msgpack und sendet sie über die Socket-Verbindung.
//
// Parameter:
//   - conn *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - data interface{}: Der zu serialisierende Datensatz.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Serialisieren oder Senden der Daten ein Problem aufgetreten ist, ansonsten nil.
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
// Die Funktion erstellt ein ChannelRequestResponse-Objekt mit dem entsprechenden Fehlergrund und sendet es über die Socket-Verbindung.
//
// Parameter:
//   - conn *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - sourceId string: Die ID der ursprünglichen Anfrage.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Senden der Antwort ein Problem aufgetreten ist, ansonsten nil.
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
// Die Funktion erstellt ein ChannelRequestResponse-Objekt mit der neuen Channel-Session-ID und sendet es über die Socket-Verbindung.
//
// Parameter:
//   - conn *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - channelRequestId string: Die ID der ursprünglichen Channel-Anfrage.
//   - channelSessionId string: Die ID der neu registrierten Channel-Sitzung.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Senden der Antwort ein Problem aufgetreten ist, ansonsten nil.
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
// Die Funktion erstellt ein ChannlSessionTransportSignal-Objekt mit dem entsprechenden Signalwert und sendet es über die Socket-Verbindung.
//
// Parameter:
//   - conn *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - channelId string: Die ID des nicht geöffneten Channels.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Senden des Signals ein Problem aufgetreten ist, ansonsten nil.
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

// channelDataTransport sendet Daten über einen bestimmten Channel und gibt die Paket-ID sowie die Größe der Daten zurück.
// Die Funktion erstellt ein ChannelSessionDataTransport-Objekt, serialisiert es und sendet es über die Socket-Verbindung.
//
// Parameter:
//   - socket *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - data []byte: Die zu sendenden Daten.
//   - channelSessionId string: Die ID der Channel-Sitzung, über die die Daten gesendet werden.
//
// Rückgabe:
//   - uint64: Die Paket-ID des gesendeten Datenpakets.
//   - int: Die Größe der gesendeten Daten in Bytes.
//   - error: Ein Fehler, falls beim Serialisieren oder Senden der Daten ein Problem aufgetreten ist, ansonsten nil.
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

// writePacketACK sendet ein ACK (Acknowledgment) für ein bestimmtes Paket über die BNG-Verbindung.
// Die Funktion erstellt ein ChannelTransportStateResponse-Objekt mit den notwendigen Informationen,
// serialisiert es und sendet es über die Socket-Verbindung des BngConn-Objekts.
//
// Parameter:
//   - conn *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - pid uint64: Die Paket-ID, für die das ACK gesendet wird.
//   - sessionId string: Die ID der aktuellen Channel-Sitzung.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Senden des ACKs ein Problem aufgetreten ist, ansonsten nil.
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

// convertAndWriteBytesIntoChan wandelt einen Go-Datensatz in transportierbare Bytes um und schreibt diese in den Schreibkanal des BngConn-Objekts.
// Die Funktion serialisiert die Daten mit msgpack und sendet sie über die Socket-Verbindung.
//
// Parameter:
//   - conn *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - data interface{}: Der zu serialisierende Datensatz.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Serialisieren oder Senden der Daten ein Problem aufgetreten ist, ansonsten nil.
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

// responseUnkownChannel sendet eine Antwort zurück, wenn ein unbekannter Kanal angefordert wurde.
// Die Funktion erstellt ein ChannelRequestResponse-Objekt mit dem entsprechenden Fehlergrund
// und sendet es über die Socket-Verbindung.
//
// Parameter:
//   - conn *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - sourceId string: Die ID der ursprünglichen Anfrage.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Senden der Antwort ein Problem aufgetreten ist, ansonsten nil.
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

// responseNewChannelSession sendet eine Antwort zurück, wenn eine neue Channel-Sitzung registriert wird.
// Die Funktion erstellt ein ChannelRequestResponse-Objekt mit der neuen Channel-Session-ID
// und sendet es über die Socket-Verbindung.
//
// Parameter:
//   - conn *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - channelRequestId string: Die ID der ursprünglichen Channel-Anfrage.
//   - channelSessionId string: Die ID der neu registrierten Channel-Sitzung.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Senden der Antwort ein Problem aufgetreten ist, ansonsten nil.
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

// responseChannelNotOpen sendet ein Signal zurück, dass der angegebene Channel nicht geöffnet ist.
// Die Funktion erstellt ein ChannlSessionTransportSignal-Objekt mit dem entsprechenden Signalwert
// und sendet es über die Socket-Verbindung.
//
// Parameter:
//   - conn *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung verwaltet.
//   - channelId string: Die ID des nicht geöffneten Channels.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Senden des Signals ein Problem aufgetreten ist, ansonsten nil.
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
