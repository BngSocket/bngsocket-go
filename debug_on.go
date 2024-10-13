//go:build debug

package bngsocket

import (
	"fmt"
	"sync"
)

var _debugMutex *sync.Mutex = new(sync.Mutex)
var debugFunc func(v ...any)

func DebugPrint(values ...interface{}) {
	_debugMutex.Lock()
	defer _debugMutex.Unlock()
	if debugFunc != nil {
		debugFunc(values...)
	} else {
		fmt.Println(values...)
	}
}

func DebugSetPrintFunction(v func(v ...any)) {
	_debugMutex.Lock()
	defer _debugMutex.Unlock()
	debugFunc = v
}
