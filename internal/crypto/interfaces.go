package crypto

type ICipherManager interface {
	SetTX(ICipher)
	AddRX(string, ICipher)
	GetTX() (ICipher, bool)
	GetRX(string) (ICipher, bool)
}

type ICipher interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}
