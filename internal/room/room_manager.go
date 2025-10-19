package room

import (
	"encoding/hex"
	"sync"
)

type roomManager struct {
	mtx   *sync.RWMutex
	rooms map[string]IRoom
}

func NewRoomManager() IRoomManager {
	return &roomManager{
		mtx:   &sync.RWMutex{},
		rooms: make(map[string]IRoom, 64),
	}
}

func (p *roomManager) Get(k []byte) (IRoom, bool) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return p.getRoom(k)
}

func (p *roomManager) Set(k []byte, r IRoom) bool {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	if _, ok := p.getRoom(k); ok {
		return false
	}

	p.rooms[encode(k)] = r
	return true
}

func (p *roomManager) Del(k []byte) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	delete(p.rooms, encode(k))
}

func (p *roomManager) getRoom(k []byte) (IRoom, bool) {
	room, ok := p.rooms[encode(k)]
	return room, ok
}

func encode(k []byte) string {
	return hex.EncodeToString(k)
}
