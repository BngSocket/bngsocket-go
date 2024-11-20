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

// _DebugPrint gibt eine Debug-Nachricht aus, wenn das Debugging aktiviert ist.
// Es verwendet einen Mutex, um einen thread-sicheren Zugriff auf die Debug-Ausgabe zu gewährleisten.
// Falls eine benutzerdefinierte Debug-Funktion gesetzt ist, wird diese aufgerufen,
// andernfalls werden die Werte mit fmt.Println ausgegeben.
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

// DebugSetPrintFunction setzt eine benutzerdefinierte Debug-Funktion.
// Diese Funktion wird verwendet, um Debug-Nachrichten auszugeben.
// Wenn keine benutzerdefinierte Funktion gesetzt ist, wird fmt.Println verwendet.
func DebugSetPrintFunction(v func(v ...any)) {
	_debugMutex.Lock()
	defer _debugMutex.Unlock()
	debugFunc = v
}
