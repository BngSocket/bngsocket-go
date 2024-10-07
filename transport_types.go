package bngsocket

type TypeInfo struct {
	Type string `msgpack:"type"`
}

type RpcDataCapsle struct {
	Type  string      `msgpack:"type"`
	Value interface{} `msgpack:"value"`
}

type RpcRequest struct {
	Type   string           `msgpack:"type"`
	Params []*RpcDataCapsle `msgpack:"parameters"`
	Name   string           `msgpack:"name"`
	Hidden bool             `msgpack:"hidden"`
	Id     string           `msgpack:"id"`
}

type RpcResponse struct {
	Type   string         `msgpack:"type"`
	Error  string         `msgpack:"error,omitempty"`
	Id     string         `msgpack:"id"`
	Return *RpcDataCapsle `msgpack:"return"`
}

type RpcHiddenFunction struct {
	FunctionId string `msgpack:"id"`
}

// Wird verwendet um eine Channel Sitzung aufzubauen
type ChannelRequest struct {
	Type               string `msgpack:"type"`
	Error              string `msgpack:"error,omitempty"`
	RequestId          string `msgpack:"id"`
	RequestedChannelId string `msgpack:"cid"`
}

// Wird verwendet um zu bestätigen oder abzulehnen
type ChannelRequestResponse struct {
	Type                string `msgpack:"type"`
	ReqId               string `msgpack:"rqid"`
	ChannelId           string `msgpack:"cid"`
	NotAcceptedByReason string `msgpack:"nabr"`
}

// Wird verwendet um Sitzungspakete zu übertragen
type ChannelSessionDataTransport struct {
	Type             string `msgpack:"type"`
	ChannelSessionId string `msgpack:"csid"`
	PackageId        uint64 `msgpack:"pid"`
	Body             []byte `msgpack:"body"`
}

// Wird verwendet um zu bestätigen das die Daten übertragen wurden
type ChannelTransportStateResponse struct {
	Type             string `msgpack:"type"`
	ChannelSessionId string `msgpack:"csid"`
	PackageId        uint64 `msgpack:"pid"`
	State            uint8  `msgpack:"state"`
}

// Wird verwendet um einen Channel Ordnungsgemäß zu schließen
type ChannlTransportSignal struct {
	Type             string `msgpack:"type"`
	ChannelSessionId string `msgpack:"csid"`
	Signal           uint64 `msgpack:"pid"`
}
