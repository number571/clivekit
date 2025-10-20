package room

type DataPacket struct {
	Type    DataType
	Ident   string
	Payload []byte
}

type DataType int

const (
	CustomDataType DataType = iota
	TextDataType
	SignalDataType
	AudioDataType
	VideoDataType

	endDataType
)
