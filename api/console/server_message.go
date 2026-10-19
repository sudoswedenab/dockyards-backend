package console

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type ServerMessageKind uint8
const (
	ServerMessageKindInvalid        ServerMessageKind = 0
	ServerMessageKindConnect        ServerMessageKind = '>'
	ServerMessageKindDisconnect 	ServerMessageKind = '<'
	ServerMessageKindInput          ServerMessageKind = '.'
)

type ServerAuthenticate struct {
}

// Sent to client when connection as a response to ClientConnect when it
// is was successful.
type ServerConnect struct {
	StreamID uint16
}

// Sent to to client when connection to a resource has been disconnected.
// This is also sent when ClientConnect fails.
type ServerDisconnect struct {
	StreamID uint16
}

// This is sent to the client for each connection when there is some input
// to be drawn.
type ServerInput struct {
	StreamID uint16
	Data     []byte
}

type ServerMessage struct {
	Authenticate ServerAuthenticate
	Connect      ServerConnect
	Disconnect   ServerDisconnect
	Input        ServerInput

	Kind ServerMessageKind
}

func ServerMessageConnect(streamID uint16) ServerMessage {
	var message ServerMessage
	message.Kind = ServerMessageKindConnect
	message.Connect.StreamID = streamID
	return message
}

func ServerMessageDisconnect(streamID uint16) ServerMessage {
	var message ServerMessage
	message.Kind = ServerMessageKindDisconnect
	message.Disconnect.StreamID = streamID
	return message
}

func ServerMessageInput(streamID uint16, data []byte) ServerMessage {
	var message ServerMessage
	message.Kind = ServerMessageKindInput
	message.Input.StreamID = streamID
	message.Input.Data = data
	return message
}

func ParseServerMessage(raw []byte) (ServerMessage, error) {
	b := bytes.NewBuffer(raw)
	var m ServerMessage
	err := binary.Read(b, binary.LittleEndian, &m.Kind)
	if err != nil {
		return ServerMessage{}, fmt.Errorf("could not parse message kind: %w", err)
	}

	switch m.Kind {
	case ServerMessageKindInvalid:
		return m, nil
	case ServerMessageKindConnect:
		err = m.Connect.decode(b)
		return m, err
	case ServerMessageKindDisconnect:
		err = m.Disconnect.decode(b)
		return m, err
	case ServerMessageKindInput:
		err = m.Input.decode(b)
		return m, err
	default:
		return ServerMessage{}, fmt.Errorf("unknown message kind: %v", m.Kind)
	}
}

func (m ServerMessage) Encode(b *bytes.Buffer) []byte {
	_ = b.WriteByte(byte(m.Kind)) // err is always nil

	switch m.Kind {
	case ServerMessageKindInvalid:
	case ServerMessageKindConnect:
		m.Connect.encode(b)
	case ServerMessageKindDisconnect:
		m.Disconnect.encode(b)
	case ServerMessageKindInput:
		m.Input.encode(b)
	default:
		panic("unhandled branch in switch")
	}

	return b.Bytes()
}

func (m *ServerAuthenticate) decode(b *bytes.Buffer) error {
	return nil
}

func (m *ServerAuthenticate) encode(b *bytes.Buffer) {
}

func (m *ServerConnect) decode(b *bytes.Buffer) error {
	return binary.Read(b, binary.LittleEndian, &m.StreamID)
}

func (m *ServerConnect) encode(b *bytes.Buffer) {
	err := binary.Write(b, binary.LittleEndian, m.StreamID) // Since bytes.Buffer Write never fails, this should also not fail
	if err != nil {
		panic(err)
	}
}

func (m *ServerDisconnect) decode(b *bytes.Buffer) error {
	return binary.Read(b, binary.LittleEndian, &m.StreamID)
}

func (m *ServerDisconnect) encode(b *bytes.Buffer) {
	err := binary.Write(b, binary.LittleEndian, m.StreamID) // Since bytes.Buffer Write never fails, this should also not fail
	if err != nil {
		panic(err)
	}
}

func (m *ServerInput) decode(b *bytes.Buffer) error {
	err := binary.Read(b, binary.LittleEndian, &m.StreamID)
	if err != nil {
		return err
	}
	m.Data = b.Bytes()
	return nil
}

func (m *ServerInput) encode(b *bytes.Buffer) {
	err := binary.Write(b, binary.LittleEndian, m.StreamID) // Since bytes.Buffer Write never fails, this should also not fail
	if err != nil {
		panic(err)
	}
	_, err = b.Write(m.Data) // err is always nil
	if err != nil {
		panic(err)
	}
}
