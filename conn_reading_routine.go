package bngsocket

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"

	"github.com/CustodiaJS/bngsocket/transport"
	"github.com/vmihailenco/msgpack/v5"
)

// Nimmt eintreffende Daten entgegen
func processReadedData(o *BngConn, data []byte) {
	// Dynamisches Unmarshallen in eine map[string]interface{} oder interface{}
	var typeInfo transport.TypeInfo
	err := msgpack.Unmarshal(data, &typeInfo)
	if err != nil {
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[0]: "+err.Error()))

		// Wird beendet
		return
	}

	// LOG
	_DebugPrint(fmt.Sprintf("BngConn(%s): Enter data: %s", o._innerhid, typeInfo.Type))

	// Dynamische Verarbeitung basierend auf dem Typ des Wertes
	switch typeInfo.Type {
	// RPC Pakete
	case "rpcreq", "rpcres":
		switch typeInfo.Type {
		case "rpcreq":
			// Der Datensatz wird als RPC Regquest eingelesen
			var rpcRequest *transport.RpcRequest
			err := msgpack.Unmarshal(data, &rpcRequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[1]: "+err.Error()))

				// Wird beendet
				return
			}

			// LOG
			_DebugPrint(fmt.Sprintf("BngConn(%s): Enter RPC-Request: %s", o._innerhid, rpcRequest.Id))

			// Das Paket wird weiterverarbeitet
			if err := processRpcRequest(o, rpcRequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[2]: "+err.Error()))

				// Wird beendet
				return
			}
		case "rpcres":
			// Der Datensatz wird als RPC Regquest eingelesen
			var rpcResponse *transport.RpcResponse
			err := msgpack.Unmarshal(data, &rpcResponse)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[3]: "+err.Error()))

				// Wird beendet
				return
			}

			// LOG
			_DebugPrint(fmt.Sprintf("BngConn(%s): Enter RPC-Response: %s", o._innerhid, rpcResponse.Id))

			// Das Paket wird weiterverarbeitet
			if err := processRpcResponse(o, rpcResponse); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[4]: "+err.Error()))

				// Wird beendet
				return
			}
		}
	// Channel Pakete
	case "chreq", "chreqresp", "chst", "chsig", "chtsr":
		switch typeInfo.Type {
		case "chreq":
			// Der Datensatz wird ChannelRequest eingelesen
			var channlrequest *transport.ChannelRequest
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[3]: "+err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelRequestPackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[4]: "+err.Error()))

				// Wird beendet
				return
			}
		case "chreqresp":
			// Der Datensatz wird als ChannelRequestResponse eingelesen
			var channlrequest *transport.ChannelRequestResponse
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[5]: "+err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelRequestResponsePackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[6]: "+err.Error()))

				// Wird beendet
				return
			}
		case "chst":
			// Der Datensatz wird als ChannelSessionDataTransport eingelesen
			var channlrequest *transport.ChannelSessionDataTransport
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[7]: "+err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelSessionPackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[8]: "+err.Error()))

				// Wird beendet
				return
			}
		case "chsig":
			// Der Datensatz wird als ChannelSessionTransportSignal eingelesen
			var channlrequest *transport.ChannlSessionTransportSignal
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[9]: "+err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelSessionSignal(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[10]: "+err.Error()))

				// Wird beendet
				return
			}
		case "chtsr":
			// Der Datensatz wird als ChannelTransportStateResponse eingelesen
			var channlrequest *transport.ChannelTransportStateResponse
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[11]: "+err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelTransportStateResponsePackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[12]: "+err.Error()))

				// Wird beendet
				return
			}
		}
	// Unbekannter Pakettyp
	default:
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		consensusProtocolTermination(o, fmt.Errorf("bngsocket->_ProcessReadedData[13]: unkown type"))

		// Wird beendet
		return
	}
}

// handleMessage liest und verarbeitet eine MSG-Nachricht aus dem Reader der BngConn.
// Die empfangenen Daten werden in den bereitgestellten Cache geschrieben. Die Funktion
// liest zunächst die Länge der Nachricht (Big-Endian), anschließend die eigentlichen
// Daten. Nach dem Schreiben in den Cache wird eine Checksumme der Daten berechnet und
// eine Debug-Ausgabe erzeugt.
//
// Parameter:
//   - cache *bytes.Buffer: Ein Puffer, in den die empfangenen Daten geschrieben werden.
//   - o *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und
//     zugehörige Ressourcen verwaltet.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Lesen oder Verarbeiten der Nachricht ein Problem
//     aufgetreten ist, ansonsten nil.s
func handleMessage(cache *bytes.Buffer, o *BngConn) error {
	// Lesen der Datenlänge (Big-Endian)
	var dataLength uint32
	if err := binary.Read(o.reader, binary.BigEndian, &dataLength); err != nil {
		return fmt.Errorf("%s: %w: %v", o._innerhid, ErrMessageLength, err)
	}

	// Lesen der Nachrichtendaten
	data := make([]byte, dataLength)
	if _, err := io.ReadFull(o.reader, data); err != nil {
		return fmt.Errorf("%s: %w: %v", o._innerhid, ErrMessageRead, err)
	}

	// Speichern der Daten im Cache
	cache.Write(data)

	// Berechne die Checksumme
	checksum := crc32.ChecksumIEEE(data)

	// Debug-Ausgabe: Checksumme und Länge der Nachricht
	_DebugPrint(fmt.Sprintf("BngConn(%s): MSG received, length=%d, checksum=%08x", o._innerhid, len(data), checksum))

	return nil
}

// handleEndTransfer verarbeitet das Ende eines Datentransfers (ET-Nachricht).
// Die Funktion berechnet die Checksumme der im Cache gespeicherten Daten und startet
// die Verarbeitung dieser Daten in einer separaten Goroutine. Nach der erfolgreichen
// Verarbeitung wird der Cache geleert.
//
// Parameter:
//   - o *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und
//     zugehörige Ressourcen verwaltet.
//   - cache *bytes.Buffer: Ein Puffer, der die empfangenen Daten enthält.
//
// Rückgabe:
//   - error: Ein Fehler, falls bei der Verarbeitung des Datensatzes ein Problem
//     aufgetreten ist, ansonsten nil.s
func handleEndTransfer(o *BngConn, cache *bytes.Buffer) error {
	// Berechne die Checksumme der Daten im Cache
	data := cache.Bytes()
	checksum := crc32.ChecksumIEEE(data)

	// Debug-Ausgabe: Länge und Checksumme der Daten
	_DebugPrint(fmt.Sprintf("BngConn(%s): ET received: Processing data (length=%d, checksum=%08x)", o._innerhid, len(data), checksum))

	// Starte die Verarbeitung in einer Goroutine
	o.backgroundProcesses.Add(1) // Informiere Wartungsgruppe
	go func(data []byte) {
		defer o.backgroundProcesses.Done() // Abschluss melden
		processReadedData(o, data)         // Interne Verarbeitung
	}(data)

	// Cache leeren
	cache.Reset()

	// Erfolgreich verarbeitet
	return nil
}

// handleACK verarbeitet eine eingehende ACK-Nachricht.
// Die Funktion liest die ACK-Daten aus dem Reader der BngConn, prüft die
// Korrektheit der Nachricht und bestätigt das ACK durch Aufruf von EnterACK.
// Bei erfolgreicher Verarbeitung wird eine Debug-Ausgabe erzeugt.
//
// Parameter:
//   - o *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und
//     zugehörige Ressourcen verwaltet.
//
// Rückgabe:
//   - error: Ein Fehler, falls beim Lesen oder Verarbeiten der ACK ein Problem
//     aufgetreten ist, ansonsten nil.s
func handleACK(o *BngConn) error {
	ack := make([]byte, 2) // Das erste 'A' wurde schon gelesen
	if _, err := io.ReadFull(o.reader, ack); err != nil {
		return fmt.Errorf("%s: %w: %v", o._innerhid, ErrACKReadFailure, err)
	}

	// Prüfen, ob die Nachricht korrekt ist
	if string(ack) != "CK" {
		return fmt.Errorf("%s: %w", o._innerhid, ErrInvalidACK)
	}

	// Rufe die Methode `EnterACK` auf, um ACK zu bestätigen
	if err := o.ackHandle.EnterACK(); err != nil {
		return fmt.Errorf("%s: failed to process ACK: %v", o._innerhid, err)
	}

	_DebugPrint(fmt.Sprintf("BngConn(%s): ACK successfully processed", o._innerhid))
	return nil
}

// constantReading führt eine kontinuierliche Leseschleife auf der Socket-Verbindung des
// gegebenen BngConn-Objekts durch. Die Funktion verarbeitet eingehende Nachrichten basierend
// auf ihrem Typ:
//   - 'M' (MSG): Ein Teil des Datensatzes wird empfangen und verarbeitet. Nach erfolgreicher
//     Verarbeitung wird eine Bestätigung (ACK) zurückgesendet.
//   - 'E' (ET): Das Ende eines Datensatzes wird empfangen. Der gesamte Datensatz aus dem
//     Cache wird verarbeitet und eine Bestätigung (ACK) wird zurückgesendet.
//   - 'A' (ACK): Eine eingehende Bestätigung wird verarbeitet.
//
// Bei Auftreten von Fehlern während des Lese- oder Verarbeitungsprozesses wird die
// Funktion `readProcessErrorHandling` aufgerufen, um den Fehler zu behandeln. Abhängig von der
// Fehlerbehandlung kann die Verbindung geschlossen oder der Cache zurückgesetzt werden, um
// den Vorgang neu zu starten.
//
// Die Schleife läuft solange, wie die Funktion `runningBackgroundServingLoop(o)` den Wert
// `true` zurückgibt. Beim Start und beim Stoppen der Leseschleife werden Debug-Informationen
// ausgegeben. Nach Beendigung der Funktion wird die Hintergrundprozesszählung
// (`o.backgroundProcesses.Done()`) aufgerufen, um den Abschluss des Lesevorgangs zu signalisieren.
//
// Parameter:
//   - o *BngConn: Ein Zeiger auf das BngConn-Objekt, das die Socket-Verbindung und zugehörige
//     Ressourcen verwaltet.s
func constantReading(o *BngConn) {
	defer func() {
		_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was stopped", o._innerhid))
		o.backgroundProcesses.Done()
	}()

	_DebugPrint(fmt.Sprintf("BngConn(%s): Constant reading from Socket was started", o._innerhid))

	var cache bytes.Buffer // Cache für MSG-Daten

	for runningBackgroundServingLoop(o) {
		// Lesen des Typ-Bytes
		msgType, err := o.reader.ReadByte()
		if err != nil {
			readProcessErrorHandling(o, err)
			break
		}

		switch msgType {
		case 'M': // MSG: Ein Teil des Datensatzes
			_DebugPrint(fmt.Sprintf("BngConn(%s): MSG received", o._innerhid))
			if err := handleMessage(&cache, o); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
			// Sende ACK zurück
			if err := writePacketACK(o); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
		case 'E': // ET: Ende des Datensatzes
			_DebugPrint(fmt.Sprintf("BngConn(%s): ET received", o._innerhid))
			// Jetzt den kompletten Datensatz aus dem Cache verarbeiten
			if err := handleEndTransfer(o, &cache); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
			// Sende ACK zurück
			if err := writePacketACK(o); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
		case 'A': // ACK: Eingehende Bestätigung
			_DebugPrint(fmt.Sprintf("BngConn(%s): ACK received", o._innerhid))
			if err := handleACK(o); err != nil {
				// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
				if readProcessErrorHandling(o, err) {
					return
				} else {
					// Der Cache wird geelert und der Vorgang wird zurückgesetzt
					cache.Reset()
					continue
				}
			}
		default:
			// Der Fehler wird verarbeitet ggf wird die Verbindung geschlossen
			if readProcessErrorHandling(o, ErrUnknownMessageType) {
				return
			} else {
				// Der Cache wird geelert und der Vorgang wird zurückgesetzt
				cache.Reset()
				continue
			}
		}
	}
}
