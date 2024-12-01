package message

import "encoding/binary"

type Message struct {
	msgLength uint32
	msgType   byte
	payload   []byte
}

// This assumes bittorrent payload
// with format: <msg length : 4 bytes><msg type : 1 byte><payload : 0 - n bytes>
func PacketPayloadToMessage(payload []byte) *Message {
	return &Message{
		msgLength: binary.LittleEndian.Uint32(payload[:4]),
		msgType:   payload[4],
		payload:   payload[5:],
	}
}

func New(msgLength uint32, msgType byte, payload []byte) *Message {
	return &Message{
		msgLength: msgLength,
		msgType:   msgType,
		payload:   payload,
	}
}
