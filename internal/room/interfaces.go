package room

import (
	"context"

	"github.com/number571/clivekit/internal/crypto"
)

type IRoomManager interface {
	Get([]byte) (IRoom, bool)
	Set([]byte, IRoom) bool
	Del([]byte)
}

type ISecureRoom interface {
	IRoom

	GetCipherManager() crypto.ICipherManager
}

type IRoom interface {
	Close()

	ReceiveDataPacket(context.Context) (*DataPacket, error)
	PublishDataPacket(context.Context, *DataPacket) error
}
