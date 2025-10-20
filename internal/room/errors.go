package room

import "errors"

var (
	ErrBuffSize      = errors.New("buff size")
	ErrGetTXCipher   = errors.New("get tx cipher")
	ErrClosedChannel = errors.New("closed channel")
)
