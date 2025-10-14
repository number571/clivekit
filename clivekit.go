package main

/*
#include <stdlib.h>
#include <string.h>

#define LIVEKIT_DESC_SIZE 16
#define LIVEKIT_BUFF_SIZE 512
#define LIVEKIT_TOPIC_SIZE 64
#define LIVEKIT_IDENT_SIZE 64

#define LIVEKIT_ERRCODE_SUCCESS     0x00
#define LIVEKIT_ERRCODE_CONNECT     0x01
#define LIVEKIT_ERRCODE_CLOSE       0x02
#define LIVEKIT_ERRCODE_GET_ROOM    0x03
#define LIVEKIT_ERRCODE_PUBLISH     0x04
#define LIVEKIT_ERRCODE_CHAN_CLOSED 0x04

typedef struct {
	char *host;
	char *api_key;
	char *api_secret;
	char *room_name;
	char *ident;
} livekit_connect_info;

typedef struct {
	char   topic[LIVEKIT_TOPIC_SIZE];
	char   ident[LIVEKIT_IDENT_SIZE];
	char   payload[LIVEKIT_BUFF_SIZE];
	size_t payload_size;
} livekit_data_packet;
*/
// #cgo LDFLAGS: -lsoxr -lopus -lopusfile
import "C"

import (
	"crypto/rand"
	"sync"
	"unsafe"

	lksdk "github.com/livekit/server-sdk-go/v2"
)

type (
	descType [C.LIVEKIT_DESC_SIZE]byte
)

var (
	gMutex = sync.RWMutex{}
	gRCMap = make(map[descType]*roomContext)
)

type roomContext struct {
	mtx  *sync.RWMutex
	desc descType
	room *lksdk.Room
	dpch chan *dataPack
}

type dataPack struct {
	topic   string
	ident   string
	payload []byte
}

//export livekit_connect_to_room
func livekit_connect_to_room(room_desc *C.char, conn_info C.livekit_connect_info) C.int {
	roomMtx := &sync.RWMutex{}
	connInfo := lksdk.ConnectInfo{
		APIKey:              C.GoString(conn_info.api_key),
		APISecret:           C.GoString(conn_info.api_secret),
		RoomName:            C.GoString(conn_info.room_name),
		ParticipantIdentity: C.GoString(conn_info.ident),
	}

	dpch := make(chan *dataPack, 2048)
	roomCallback := &lksdk.RoomCallback{
		ParticipantCallback: lksdk.ParticipantCallback{
			OnDataPacket: onDataPacketCallback(roomMtx, dpch),
		},
	}

	room, err := lksdk.ConnectToRoom(C.GoString(conn_info.host), connInfo, roomCallback)
	if err != nil {
		return C.LIVEKIT_ERRCODE_CONNECT
	}

	return createRoomContext(room_desc, roomMtx, room, dpch)
}

//export livekit_disconnect_from_room
func livekit_disconnect_from_room(room_desc *C.char) C.int {
	if ok := closeRoomContextByDesc(room_desc); !ok {
		return C.LIVEKIT_ERRCODE_CLOSE
	}
	return C.LIVEKIT_ERRCODE_SUCCESS
}

//export livekit_read_data_from_room
func livekit_read_data_from_room(room_desc *C.char, data_packet *C.livekit_data_packet) C.int {
	rc, ok := getRoomContextByDesc(room_desc)
	if !ok {
		return C.LIVEKIT_ERRCODE_GET_ROOM
	}

	td, ok := <-rc.dpch
	if !ok {
		return C.LIVEKIT_ERRCODE_CHAN_CLOSED
	}

	cPayloadSize := C.size_t(len(td.payload))

	cIdent := C.CString(td.ident)
	cTopic := C.CString(td.topic)
	defer C.free(unsafe.Pointer(cIdent))
	defer C.free(unsafe.Pointer(cTopic))

	data_packet.payload_size = cPayloadSize
	C.strncpy(&data_packet.topic[0], cTopic, C.size_t(len(td.topic)+1))
	C.strncpy(&data_packet.ident[0], cIdent, C.size_t(len(td.ident)+1))
	C.memcpy(unsafe.Pointer(&data_packet.payload), unsafe.Pointer(&td.payload[0]), cPayloadSize)

	return C.LIVEKIT_ERRCODE_SUCCESS
}

//export livekit_write_data_to_room
func livekit_write_data_to_room(room_desc, topic, data *C.char, data_size C.int) C.int {
	rc, ok := getRoomContextByDesc(room_desc)
	if !ok {
		return C.LIVEKIT_ERRCODE_GET_ROOM
	}

	goData := C.GoBytes(unsafe.Pointer(data), data_size)
	err := rc.room.LocalParticipant.PublishDataPacket(
		lksdk.UserData(goData),
		lksdk.WithDataPublishTopic(C.GoString(topic)),
	)
	if err != nil {
		return C.LIVEKIT_ERRCODE_PUBLISH
	}

	return C.LIVEKIT_ERRCODE_SUCCESS
}

func onDataPacketCallback(roomMtx *sync.RWMutex, dpch chan *dataPack) func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
	return func(data lksdk.DataPacket, params lksdk.DataReceiveParams) {
		dp, ok := data.(*lksdk.UserDataPacket)
		if !ok {
			return
		}
		offset := C.LIVEKIT_BUFF_SIZE
		pldLength := len(dp.Payload)
		for i := 0; i < pldLength; i += offset {
			end := i + offset
			if end > pldLength {
				end = pldLength
			}
			roomMtx.Lock()
			select {
			case dpch <- &dataPack{
				topic:   dp.Topic,
				ident:   params.SenderIdentity,
				payload: dp.Payload[i:end],
			}:
			default:
			}
			roomMtx.Unlock()
		}
	}
}

func createRoomContext(cRoomDesc *C.char, mtx *sync.RWMutex, room *lksdk.Room, dpch chan *dataPack) C.int {
	goRoomDesc := genGoDesc()
	gMutex.Lock()
	gRCMap[goRoomDesc] = &roomContext{
		mtx:  mtx,
		desc: goRoomDesc,
		room: room,
		dpch: dpch,
	}
	gMutex.Unlock()
	setGoDesc(cRoomDesc, goRoomDesc)
	return C.LIVEKIT_ERRCODE_SUCCESS
}

func getRoomContextByDesc(cRoomDesc *C.char) (*roomContext, bool) {
	gMutex.RLock()
	v, ok := gRCMap[getGoDesc(cRoomDesc)]
	gMutex.RUnlock()
	return v, ok
}

func closeRoomContextByDesc(cRoomDesc *C.char) bool {
	rc, ok := getRoomContextByDesc(cRoomDesc)
	if !ok {
		return false
	}

	gMutex.Lock()
	rc.mtx.Lock()

	rc.room.Disconnect()
	delete(gRCMap, rc.desc)
	close(rc.dpch)

	rc.mtx.Unlock()
	gMutex.Unlock()

	return true
}

func setGoDesc(cDesc *C.char, goDesc descType) {
	C.memcpy(unsafe.Pointer(cDesc), unsafe.Pointer(&goDesc[0]), C.LIVEKIT_DESC_SIZE)
}

func getGoDesc(cDesc *C.char) descType {
	var goDesc descType
	loadDesc := C.GoBytes(unsafe.Pointer(cDesc), C.LIVEKIT_DESC_SIZE)
	copy(goDesc[:], loadDesc)
	return goDesc
}

func genGoDesc() descType {
	var desc descType
	_, _ = rand.Read(desc[:])
	return desc
}

func main() {
	// Required for cgo to work, even if empty
}
