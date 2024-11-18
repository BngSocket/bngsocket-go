package bngsocket

import (
	"errors"
	"strings"
)

var (
	ErrConcurrentReadingNotAllowed = errors.New("concurrent reading is not allowed")
	ErrConcurrentWritingNotAllowed = errors.New("concurrent writing is not allowed")
	ErrConnectionClosedEOF         = errors.New("connection closed EOF")
	ErrUnkownRpcFunction           = errors.New("unkown rpc function called")
)

// Wandelt einen Fehler welcher mittels String übertragen wurde zurück in einen Go Fehler
func processError(errString string) error {
	switch {
	case strings.Contains(ErrUnkownRpcFunction.Error(), errString):
		return ErrUnkownRpcFunction
	default:
		return errors.New(errString)
	}
}
