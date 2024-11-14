package bngsocket

import "errors"

var ErrConcurrentReadingNotAllowed = errors.New("concurrent reading is not allowed")
var ErrConcurrentWritingNotAllowed = errors.New("concurrent writing is not allowed")
var ErrConnectionClosedEOF = errors.New("connection closed EOF")
