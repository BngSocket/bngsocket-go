package bngsocket

import (
	"bufio"
	"net"
	"reflect"
	"sync"

	"github.com/CustodiaJS/bngsocket/transport"
	"github.com/google/uuid"
)

func _NewBaseBngSocketObject(socket net.Conn) *BngConn {
	bngConn := &BngConn{
		conn:                     socket,
		writer:                   bufio.NewWriter(socket),
		reader:                   bufio.NewReader(socket),
		connMutex:                new(sync.Mutex),
		_innerhid:                uuid.NewString(),
		ackHandle:                newConnACK(),
		backgroundProcesses:      &sync.WaitGroup{},
		closed:                   newSafeBool(false),
		closing:                  newSafeBool(false),
		writerMutex:              new(sync.Mutex),
		functions:                newSafeMap[string, reflect.Value](),
		hiddenFunctions:          newSafeMap[string, reflect.Value](),
		openRpcRequests:          _SafeMap[string, chan *transport.RpcResponse]{Map: new(sync.Map)},
		openChannelListener:      newSafeMap[string, *BngConnChannelListener](),
		openChannelInstances:     newSafeMap[string, *BngConnChannel](),
		openChannelJoinProcesses: _SafeMap[string, chan *transport.ChannelRequestResponse]{Map: new(sync.Map)},
		runningError:             newSafeValue[error](nil),
	}
	return bngConn
}
