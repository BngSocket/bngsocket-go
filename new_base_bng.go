package bngsocket

import (
	"reflect"
	"sync"

	"github.com/google/uuid"
)

func _NewBaseBngSocketObject() *BngConn {
	// Das BngConn Objekt wird erzeugt
	return &BngConn{
		_innerhid:                uuid.NewString(),
		backgroundProcesses:      &sync.WaitGroup{},
		mu:                       &sync.Mutex{},
		closed:                   newSafeBool(false),
		closing:                  newSafeBool(false),
		writingChan:              NewBufferdSafeChan[*dataWritingResolver](64000),
		functions:                newSafeMap[string, reflect.Value](),
		hiddenFunctions:          newSafeMap[string, reflect.Value](),
		openRpcRequests:          SafeMap[string, chan *RpcResponse]{Map: new(sync.Map)},
		openChannelListener:      newSafeMap[string, *BngConnChannelListener](),
		openChannelInstances:     newSafeMap[string, *BngConnChannel](),
		openChannelJoinProcesses: SafeMap[string, chan *ChannelRequestResponse]{Map: new(sync.Map)},
		runningError:             newSafeValue[error](nil),
	}
}
