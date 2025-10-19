package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type sCipher struct {
	block cipher.Block
}

func NewCipher(key []byte) ICipher {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	return &sCipher{block}
}

func (p *sCipher) Encrypt(plaintext []byte) ([]byte, error) {
	gcm, err := cipher.NewGCM(p.block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil) // nonce is prepended to ciphertext
	return ciphertext, nil
}

func (p *sCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	gcm, err := cipher.NewGCM(p.block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
