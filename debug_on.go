//go:build debug

package bngsocket

import "fmt"

func DebugPrint(values ...interface{}) {
	fmt.Println(values...)
}

func SetPrintFunction(v func(v ...any)) {

}
