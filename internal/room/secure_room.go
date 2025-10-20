package room

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/number571/clivekit/internal/crypto"
)

var (
	_ ISecureRoom = &secureRoom{}
)

type secureRoom struct {
	mtx           *sync.RWMutex
	lksdkRoom     *lksdk.Room
	closed        chan struct{}
	dataPackCh    chan *DataPacket
	cipherManager crypto.ICipherManager
}

type ConnectInfo struct {
	Host     string
	BuffSize int
	lksdk.ConnectInfo
}

func ConnectToSecureRoom(connInfo *ConnectInfo) (ISecureRoom, error) {
	var (
		mtx           = &sync.RWMutex{}
		closed        = make(chan struct{})
		cipherManager = crypto.NewCipherManager()
		dataPackCh    = make(chan *DataPacket, 2048)
	)

	roomCallback := &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnDataPacket: onDataPacketCallback(mtx, closed, connInfo.BuffSize, cipherManager, dataPackCh),
		},
	}

	lksdkRoom, err := lksdk.ConnectToRoom(connInfo.Host, connInfo.ConnectInfo, roomCallback)
	if err != nil {
		return nil, err
	}

	return &secureRoom{
		mtx:           mtx,
		closed:        closed,
		lksdkRoom:     lksdkRoom,
		dataPackCh:    dataPackCh,
		cipherManager: cipherManager,
	}, nil
}

func (p *secureRoom) GetCipherManager() crypto.ICipherManager {
	return p.cipherManager
}

func (p *secureRoom) Close() {
	for {
		if ok := p.mtx.TryLock(); ok {
			defer p.mtx.Unlock()
			break
		}
		select {
		case <-p.dataPackCh:
		default:
		}
	}
	close(p.closed)
	close(p.dataPackCh)
	p.lksdkRoom.Disconnect()
}

func (p *secureRoom) ReceiveDataPacket(ctx context.Context) (*DataPacket, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case dp, ok := <-p.dataPackCh:
		if !ok {
			return nil, ErrClosedChannel
		}
		return dp, nil
	}
}

func (p *secureRoom) PublishDataPacket(_ context.Context, dataPack *DataPacket) error {
	cipher, ok := p.cipherManager.GetTX()
	if !ok {
		return ErrGetTXCipher
	}
	encData, err := cipher.Encrypt(dataPack.Payload)
	if err != nil {
		return err
	}
	isReliable := (dataPack.Type == TextDataType) || (dataPack.Type == SignalDataType)
	return p.lksdkRoom.LocalParticipant.PublishDataPacket(
		lksdk.UserData(encData),
		lksdk.WithDataPublishTopic(fmt.Sprintf("%d", dataPack.Type)),
		lksdk.WithDataPublishReliable(isReliable),
	)
}

func onDataPacketCallback(
	roomMtx *sync.RWMutex,
	closed chan struct{},
	buffSize int,
	cipherManager crypto.ICipherManager,
	dataPackCh chan *DataPacket,
) func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
	return func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
		dp, ok := data.(*lksdk.UserDataPacket)
		if !ok {
			return
		}
		ident := params.SenderIdentity
		cipher, ok := cipherManager.GetRX(ident)
		if !ok {
			return
		}
		decPld, err := cipher.Decrypt(dp.Payload)
		if err != nil {
			return
		}
		dataType, err := strconv.Atoi(dp.Topic)
		if err != nil || dataType >= int(endDataType) {
			return
		}
		pldSize := len(decPld)
		for i := 0; i < pldSize; i += buffSize {
			end := i + buffSize
			if end > pldSize {
				end = pldSize
			}
			roomMtx.Lock()
			select {
			case <-closed:
				return
			default:
			}
			select {
			case dataPackCh <- &DataPacket{
				Type:    DataType(dataType),
				Ident:   ident,
				Payload: decPld[i:end],
			}:
			default:
			}
			roomMtx.Unlock()
		}
	}
}
