//go:build !debug

package bngsocket

func _DebugPrint(values ...interface{}) {
}

func DebugSetPrintFunction(v func(v ...any)) {
}
