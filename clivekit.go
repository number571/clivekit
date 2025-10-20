package main

/*
#include <stdlib.h>
#include <string.h>

#define CLIVEKIT_SIZE_DESC   16
#define CLIVEKIT_SIZE_BUFF   512
#define CLIVEKIT_SIZE_IDENT  32
#define CLIVEKIT_SIZE_ENCKEY 32 // 256-bit key

typedef enum {
	CLIVEKIT_ETYPE_SUCCESS,
	CLIVEKIT_ETYPE_CONNECT,
	CLIVEKIT_ETYPE_CLOSE,
	CLIVEKIT_ETYPE_GET_ROOM,
	CLIVEKIT_ETYPE_PUBLISH,
	CLIVEKIT_ETYPE_RECEIVE,
	CLIVEKIT_ETYPE_CREATE_ROOM
} clivekit_error_type;

typedef enum {
	CLIVEKIT_DTYPE_CUSTOM,
	CLIVEKIT_DTYPE_TEXT,
	CLIVEKIT_DTYPE_SIGNAL,
	CLIVEKIT_DTYPE_AUDIO,
	CLIVEKIT_DTYPE_VIDEO
} clivekit_data_type;

typedef struct {
	char *host;
	char *api_key;
	char *api_secret;
	char *room_name;
	char *ident;
} clivekit_connect_info;

typedef struct {
	clivekit_data_type dtype;
	char              ident[CLIVEKIT_SIZE_IDENT];
	char              payload[CLIVEKIT_SIZE_BUFF];
	size_t            payload_size;
} clivekit_data_packet;
*/
// #cgo LDFLAGS: -lsoxr -lopus -lopusfile
import "C"

import (
	"context"
	"crypto/rand"
	"unsafe"

	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/number571/clivekit/internal/crypto"
	"github.com/number571/clivekit/internal/room"
)

type (
	descType [C.CLIVEKIT_SIZE_DESC]byte
)

var (
	roomManager = room.NewRoomManager()
)

//export clivekit_connect_to_room
func clivekit_connect_to_room(room_desc *C.char, conn_info C.clivekit_connect_info) C.clivekit_error_type {
	connInfo := &room.ConnectInfo{
		Host:     C.GoString(conn_info.host),
		BuffSize: C.CLIVEKIT_SIZE_BUFF,
		ConnectInfo: lksdk.ConnectInfo{
			APIKey:              C.GoString(conn_info.api_key),
			APISecret:           C.GoString(conn_info.api_secret),
			RoomName:            C.GoString(conn_info.room_name),
			ParticipantIdentity: C.GoString(conn_info.ident),
		},
	}

	room, err := room.ConnectToSecureRoom(connInfo)
	if err != nil {
		return C.CLIVEKIT_ETYPE_CONNECT
	}

	if ok := createRoomContext(room_desc, room); !ok {
		room.Close()
		return C.CLIVEKIT_ETYPE_CREATE_ROOM
	}

	return C.CLIVEKIT_ETYPE_SUCCESS
}

//export clivekit_disconnect_from_room
func clivekit_disconnect_from_room(room_desc *C.char) C.clivekit_error_type {
	if ok := closeRoomContextByDesc(room_desc); !ok {
		return C.CLIVEKIT_ETYPE_CLOSE
	}
	return C.CLIVEKIT_ETYPE_SUCCESS
}

//export clivekit_add_rx_key_for_room
func clivekit_add_rx_key_for_room(room_desc, ident, rx_key *C.char) C.clivekit_error_type {
	rc, _, ok := getRoomContextByDesc(room_desc)
	if !ok {
		return C.CLIVEKIT_ETYPE_GET_ROOM
	}

	key := C.GoBytes(unsafe.Pointer(rx_key), C.CLIVEKIT_SIZE_ENCKEY)
	rc.GetCipherManager().AddRX(C.GoString(ident), crypto.NewCipher(key))
	return C.CLIVEKIT_ETYPE_SUCCESS
}

//export clivekit_del_rx_key_for_room
func clivekit_del_rx_key_for_room(room_desc, ident *C.char) C.clivekit_error_type {
	rc, _, ok := getRoomContextByDesc(room_desc)
	if !ok {
		return C.CLIVEKIT_ETYPE_GET_ROOM
	}

	rc.GetCipherManager().DelRX(C.GoString(ident))
	return C.CLIVEKIT_ETYPE_SUCCESS
}

//export clivekit_set_tx_key_for_room
func clivekit_set_tx_key_for_room(room_desc, tx_key *C.char) C.clivekit_error_type {
	rc, _, ok := getRoomContextByDesc(room_desc)
	if !ok {
		return C.CLIVEKIT_ETYPE_GET_ROOM
	}

	key := C.GoBytes(unsafe.Pointer(tx_key), C.CLIVEKIT_SIZE_ENCKEY)
	rc.GetCipherManager().SetTX(crypto.NewCipher(key))
	return C.CLIVEKIT_ETYPE_SUCCESS
}

//export clivekit_read_data_from_room
func clivekit_read_data_from_room(room_desc *C.char, data_packet *C.clivekit_data_packet) C.clivekit_error_type {
	rc, _, ok := getRoomContextByDesc(room_desc)
	if !ok {
		return C.CLIVEKIT_ETYPE_GET_ROOM
	}

	td, err := rc.ReceiveDataPacket(context.Background())
	if err != nil {
		return C.CLIVEKIT_ETYPE_RECEIVE
	}

	cPayloadSize := C.size_t(len(td.Payload))
	cIdent := C.CString(td.Ident)
	defer C.free(unsafe.Pointer(cIdent))

	data_packet.dtype = C.clivekit_data_type(td.Type)
	data_packet.payload_size = cPayloadSize
	C.memcpy(unsafe.Pointer(&data_packet.ident), unsafe.Pointer(cIdent), C.size_t(len(td.Ident)+1))
	C.memcpy(unsafe.Pointer(&data_packet.payload), unsafe.Pointer(&td.Payload[0]), cPayloadSize)

	return C.CLIVEKIT_ETYPE_SUCCESS
}

//export clivekit_write_data_to_room
func clivekit_write_data_to_room(room_desc *C.char, data_type C.clivekit_data_type, data *C.char, data_size C.size_t) C.clivekit_error_type {
	ctx := context.Background()

	rc, _, ok := getRoomContextByDesc(room_desc)
	if !ok {
		return C.CLIVEKIT_ETYPE_GET_ROOM
	}

	var (
		fullPayload = C.GoBytes(unsafe.Pointer(data), C.int(data_size))
		fullPldSize = uint64(data_size)
		sDataPacket = &room.DataPacket{Type: convertDataType(data_type)}
	)

	for i := uint64(0); i < fullPldSize; i += C.CLIVEKIT_SIZE_BUFF {
		end := i + C.CLIVEKIT_SIZE_BUFF
		if end > fullPldSize {
			end = fullPldSize
		}
		sDataPacket.Payload = fullPayload[i:end]
		if err := rc.PublishDataPacket(ctx, sDataPacket); err != nil {
			return C.CLIVEKIT_ETYPE_PUBLISH
		}
	}

	return C.CLIVEKIT_ETYPE_SUCCESS
}

func convertDataType(data_type C.clivekit_data_type) room.DataType {
	switch data_type {
	case C.CLIVEKIT_DTYPE_CUSTOM:
		return room.CustomDataType
	case C.CLIVEKIT_DTYPE_TEXT:
		return room.TextDataType
	case C.CLIVEKIT_DTYPE_SIGNAL:
		return room.SignalDataType
	case C.CLIVEKIT_DTYPE_AUDIO:
		return room.AudioDataType
	case C.CLIVEKIT_DTYPE_VIDEO:
		return room.VideoDataType
	}
	panic("unknown data type")
}

func createRoomContext(cRoomDesc *C.char, room room.ISecureRoom) bool {
	var goRoomDesc descType
	if _, err := rand.Read(goRoomDesc[:]); err != nil {
		return false
	}
	if ok := roomManager.Set(goRoomDesc[:], room); !ok {
		return false
	}
	C.memcpy(unsafe.Pointer(cRoomDesc), unsafe.Pointer(&goRoomDesc[0]), C.CLIVEKIT_SIZE_DESC)
	return true
}

func getRoomContextByDesc(cRoomDesc *C.char) (room.ISecureRoom, descType, bool) {
	var goRoomDesc descType
	loadDesc := C.GoBytes(unsafe.Pointer(cRoomDesc), C.CLIVEKIT_SIZE_DESC)
	copy(goRoomDesc[:], loadDesc)
	v, ok := roomManager.Get(goRoomDesc[:])
	return v.(room.ISecureRoom), goRoomDesc, ok
}

func closeRoomContextByDesc(cRoomDesc *C.char) bool {
	rc, desc, ok := getRoomContextByDesc(cRoomDesc)
	if !ok {
		return false
	}
	roomManager.Del(desc[:])
	rc.Close()
	return true
}

func main() {
	// Required for cgo to work, even if empty
}
