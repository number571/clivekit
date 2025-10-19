package room

import "errors"

var (
	ErrGetTXCipher   = errors.New("get tx cipher")
	ErrClosedChannel = errors.New("closed channel")
)
