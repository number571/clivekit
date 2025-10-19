package crypto

import (
	"sync"
)

type cipherManager struct {
	mtx *sync.RWMutex
	tx  ICipher
	rxs map[string]ICipher
}

func NewCipherManager() ICipherManager {
	return &cipherManager{
		mtx: &sync.RWMutex{},
		rxs: make(map[string]ICipher, 64),
	}
}

func (p *cipherManager) SetTX(v ICipher) {
	p.mtx.Lock()
	p.tx = v
	p.mtx.Unlock()
}

func (p *cipherManager) AddRX(k string, v ICipher) {
	p.mtx.Lock()
	p.rxs[k] = v
	p.mtx.Unlock()
}

func (p *cipherManager) DelRX(k string) {
	p.mtx.Lock()
	delete(p.rxs, k)
	p.mtx.Unlock()
}

func (p *cipherManager) GetTX() (ICipher, bool) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	return p.tx, p.tx != nil
}

func (p *cipherManager) GetRX(k string) (ICipher, bool) {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	cipher, ok := p.rxs[k]
	return cipher, ok
}
