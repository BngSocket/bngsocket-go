package bngsocket

import (
	"flag"
	"fmt"
	"sync"
)

var _debugMutex *sync.Mutex = new(sync.Mutex)
var debugFunc func(v ...any)

var (
	debugEnable = flag.Bool("bng_debug", false, "Debug modus")
)

func _DebugPrint(values ...interface{}) {
	if *debugEnable {
		_debugMutex.Lock()
		defer _debugMutex.Unlock()
		if debugFunc != nil {
			debugFunc(values...)
		} else {
			fmt.Println(values...)
		}
	}
}

func DebugSetPrintFunction(v func(v ...any)) {
	_debugMutex.Lock()
	defer _debugMutex.Unlock()
	debugFunc = v
}
