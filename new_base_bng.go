package bngsocket

import (
	"bufio"
	"net"
	"reflect"
	"sync"

	"github.com/google/uuid"
)

func _NewBaseBngSocketObject(socket net.Conn) *BngConn {
	// Das BngConn Objekt wird erzeugt
	return &BngConn{
		conn:                     socket,
		connMutex:                new(sync.Mutex),
		_innerhid:                uuid.NewString(),
		backgroundProcesses:      &sync.WaitGroup{},
		closed:                   newSafeBool(false),
		closing:                  newSafeBool(false),
		writer:                   bufio.NewWriter(socket),
		writerMutex:              new(sync.Mutex),
		functions:                newSafeMap[string, reflect.Value](),
		hiddenFunctions:          newSafeMap[string, reflect.Value](),
		openRpcRequests:          _SafeMap[string, chan *RpcResponse]{Map: new(sync.Map)},
		openChannelListener:      newSafeMap[string, *BngConnChannelListener](),
		openChannelInstances:     newSafeMap[string, *BngConnChannel](),
		openChannelJoinProcesses: _SafeMap[string, chan *ChannelRequestResponse]{Map: new(sync.Map)},
		runningError:             newSafeValue[error](nil),
	}
}
