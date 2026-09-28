package webshell

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=ServerMessageKind
type ServerMessageKind uint8
const (
	ServerMessageKindInvalid        ServerMessageKind = 0
	ServerMessageKindAuthenticate   ServerMessageKind = 1
	ServerMessageKindInput          ServerMessageKind = 2
)

type ServerMessage struct {
	Input struct {
		Data []byte
	}

	Kind ServerMessageKind
}

func ParseServerMessage(raw []byte) (ServerMessage, error) {
	buf := bytes.NewBuffer(raw)
	var m ServerMessage
	err := binary.Read(buf, binary.LittleEndian, &m.Kind)
	if err != nil {
		return ServerMessage{}, fmt.Errorf("could not parse message kind: %w", err)
	}

	switch m.Kind {
	case ServerMessageKindInvalid:
		return m, nil
	case ServerMessageKindAuthenticate:
		return m, nil
	case ServerMessageKindInput:
		m.Input.Data = buf.Bytes()
	default:
		return ServerMessage{}, fmt.Errorf("unknown message kind: %v", m.Kind)
	}

	return m, nil
}

func ServerMessageAuthenticate() ServerMessage {
	var message ServerMessage
	message.Kind = ServerMessageKindAuthenticate
	return message
}

func ServerMessageInput(data []byte) ServerMessage {
	var message ServerMessage
	message.Kind = ServerMessageKindInput
	message.Input.Data = data
	return message
}

func (m ServerMessage) Encode(b *bytes.Buffer) []byte {
	_ = b.WriteByte(byte(m.Kind)) // err is always nil

	switch m.Kind {
	case ServerMessageKindInvalid:
		break
	case ServerMessageKindAuthenticate:
		break
	case ServerMessageKindInput:
		_, _ = b.Write(m.Input.Data) // err is always nil
	default:
		panic("unhandled branch in switch")
	}

	return b.Bytes()
}
