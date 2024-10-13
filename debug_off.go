//go:build !debug

package bngsocket

func DebugPrint(values ...interface{}) {
}

func SetPrintFunction(v func(v ...any)) {
}
