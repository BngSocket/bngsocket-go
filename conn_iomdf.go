package bngsocket

import (
	"fmt"

	"github.com/vmihailenco/msgpack/v5"
)

// Nimmt eintreffende Daten entgegen
func (o *BngConn) _ProcessReadedData(data []byte) {
	// Dynamisches Unmarshallen in eine map[string]interface{} oder interface{}
	var typeInfo TypeInfo
	err := msgpack.Unmarshal(data, &typeInfo)
	if err != nil {
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[0]: " + err.Error()))

		// Wird beendet
		return
	}

	// Dynamische Verarbeitung basierend auf dem Typ des Wertes
	switch typeInfo.Type {
	// RPC Pakete
	case "rpcreq", "rpcres":
		switch typeInfo.Type {
		case "rpcreq":
			// Der Datensatz wird als RPC Regquest eingelesen
			var rpcRequest *RpcRequest
			err := msgpack.Unmarshal(data, &rpcRequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[1]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := processRpcRequest(o, rpcRequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[2]: " + err.Error()))

				// Wird beendet
				return
			}
		case "rpcres":
			// Der Datensatz wird als RPC Regquest eingelesen
			var rpcResponse *RpcResponse
			err := msgpack.Unmarshal(data, &rpcResponse)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[3]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := processRpcResponse(o, rpcResponse); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[4]: " + err.Error()))

				// Wird beendet
				return
			}
		}
	// Channel Pakete
	case "chreq", "chreqresp", "chst", "chsig", "chtsr":
		switch typeInfo.Type {
		case "chreq":
			// Der Datensatz wird ChannelRequest eingelesen
			var channlrequest *ChannelRequest
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[3]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelRequestPackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[4]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chreqresp":
			// Der Datensatz wird als ChannelRequestResponse eingelesen
			var channlrequest *ChannelRequestResponse
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[5]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelRequestResponsePackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[6]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chst":
			// Der Datensatz wird als ChannelSessionDataTransport eingelesen
			var channlrequest *ChannelSessionDataTransport
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[7]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelSessionPackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[8]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chsig":
			// Der Datensatz wird als ChannelSessionTransportSignal eingelesen
			var channlrequest *ChannlSessionTransportSignal
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[9]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelSessionSignal(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[10]: " + err.Error()))

				// Wird beendet
				return
			}
		case "chtsr":
			// Der Datensatz wird als ChannelTransportStateResponse eingelesen
			var channlrequest *ChannelTransportStateResponse
			err := msgpack.Unmarshal(data, &channlrequest)
			if err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[11]: " + err.Error()))

				// Wird beendet
				return
			}

			// Das Paket wird weiterverarbeitet
			if err := o._ProcessIncommingChannelTransportStateResponsePackage(channlrequest); err != nil {
				// Aus Sicherheitsgründen wird die Verbindung terminiert
				o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[12]: " + err.Error()))

				// Wird beendet
				return
			}
		}
	// Unbekannter Pakettyp
	default:
		// Aus Sicherheitsgründen wird die Verbindung terminiert
		o._ConsensusProtocolTermination(fmt.Errorf("bngsocket->_ProcessReadedData[13]: unkown type"))

		// Wird beendet
		return
	}
}

// Wird verwendet wenn ein Abweichender Protokoll Fehler auftritt
func (o *BngConn) _ConsensusProtocolTermination(reason error) {
	// Es wird geprüft ob die Verbindung bereits geschlossen wurde, wenn ja, wird der Vorgang verworfen
	if connectionIsClosed(o) {
		return
	}

	// Der connMutextex wird angewenet
	o.connMutex.Lock()
	defer o.connMutex.Unlock()

	// Der Fehler wird geschrieben
	if reason == nil {
		o.runningError.Set(fmt.Errorf(""))
	} else {
		o.runningError.Set(reason)
	}

	// Es wird Signalisiert dass die Verbindung geschlossen wurde
	o.closed.Set(true)

	// Die Socket Verbindung wird geschlossen
	o.conn.Close()
}
