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
	ErrInvalidACK                  = errors.New("invalid ACK received")
	ErrACKReadFailure              = errors.New("failed to read ACK")
	ErrMessageLength               = errors.New("failed to read message length")
	ErrMessageRead                 = errors.New("failed to read message data")
	ErrConnectionClosed            = errors.New("connection closed")
	ErrReadMessageType             = errors.New("failed to read message type")
	ErrProcessingMessage           = errors.New("failed to process MSG")
	ErrProcessingEndTransfer       = errors.New("failed to process ET")
	ErrProcessingACK               = errors.New("failed to process ACK")
	ErrUnknownMessageType          = errors.New("unknown message type")
	ErrWriteMessageType            = errors.New("failed to write message type")
	ErrWriteChunkLength            = errors.New("failed to write chunk length")
	ErrWriteChunk                  = errors.New("failed to write chunk data")
	ErrFlushWriter                 = errors.New("failed to flush writer")
	ErrWriteEndTransfer            = errors.New("failed to write end transfer")
	ErrWaitForACK                  = errors.New("failed to wait for ACK")
	ErrWriteACK                    = errors.New("failed to write ACK")
	ErrFlushACK                    = errors.New("failed to flush ACK writer")
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
