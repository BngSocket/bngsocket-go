//go:build !debug

package bngsocket

func DebugPrint(values ...interface{}) {
}

func DebugSetPrintFunction(v func(v ...any)) {
}
