package bngsocket

// Definiere einen Struct nur für das "type"-Feld
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
