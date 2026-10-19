package console

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type ClientMessageKind uint8
const (
	ClientMessageKindInvalid        ClientMessageKind = 0
	ClientMessageKindConnect        ClientMessageKind = '>'
	ClientMessageKindDisconnect 	ClientMessageKind = '<'
	ClientMessageKindInput          ClientMessageKind = '.'
	ClientMessageKindResize         ClientMessageKind = '='
)

// This is sent to the server when start using a resource.
// The server responds with either ServerConnect or ServerDisconnect.
type ClientConnect struct {
	// Console multiplexes multiple sessions through the same websocket connection
	// This field is used as a way of distinguishing the different streams.
	// The client is responsible for choosing a unique identifier.
	// It is erronious to call connect with the same ID twice.
	// It is also erronious to use different connection IDs for the same resource name.
	StreamID     uint16
	ResourceName string // namespace/name
}

// This is sent to the server when we want to stop using a resource.
type ClientDisconnect struct {
	StreamID uint16
}

// This is sent to the server when there is keyboard activity.
type ClientInput struct {
	StreamID uint16
	Data     []byte
}

// This is sent to the server when the console window is resized.
type ClientResize struct {
	StreamID uint16
	Rows     uint16
	Cols     uint16
	XPixel   uint16 // width in pixels
	YPixel   uint16 // height in pixels
}

type ClientMessage struct {
	Connect      ClientConnect
	Disconnect   ClientDisconnect
	Input        ClientInput
	Resize       ClientResize

	Kind ClientMessageKind
}

func (m *ClientMessage) StreamID() uint16 {
	switch (m.Kind) {
	case ClientMessageKindInvalid:
		return 0
	case ClientMessageKindConnect:
		return m.Connect.StreamID
	case ClientMessageKindDisconnect:
		return m.Disconnect.StreamID
	case ClientMessageKindInput:
		return m.Input.StreamID
	case ClientMessageKindResize:
		return m.Resize.StreamID
	default:
		panic("non-exhaustive switch")
	}
}

func ClientMessageConnect(streamID uint16, resourceName string) ClientMessage {
	var m ClientMessage
	m.Kind = ClientMessageKindConnect
	m.Connect.StreamID = streamID
	m.Connect.ResourceName = resourceName
	return m
}

func ClientMessageDisconnect(streamID uint16, resourceName string) ClientMessage {
	var m ClientMessage
	m.Kind = ClientMessageKindDisconnect
	m.Disconnect.StreamID = streamID
	return m
}

func ClientMessageInput(streamID uint16, data []byte) ClientMessage {
	var m ClientMessage
	m.Kind = ClientMessageKindInput
	m.Input.StreamID = streamID
	m.Input.Data = data
	return m
}

func ClientMessageResize(streamID uint16, rows uint16, cols uint16, xpixel uint16, ypixel uint16) ClientMessage {
	var m ClientMessage
	m.Kind = ClientMessageKindResize
	m.Resize.StreamID = streamID
	m.Resize.Rows = rows
	m.Resize.Cols = cols
	m.Resize.XPixel = xpixel
	m.Resize.YPixel = ypixel
	return m
}

func DecodeClientMessage(raw []byte) (ClientMessage, error) {
	b := bytes.NewBuffer(raw)
	var m ClientMessage
	err := binary.Read(b, binary.LittleEndian, &m.Kind)
	if err != nil {
		return ClientMessage{}, fmt.Errorf("could not parse message kind: %w", err)
	}

	switch m.Kind {
	case ClientMessageKindInvalid:
		return m, nil
	case ClientMessageKindConnect:
		err = m.Connect.decode(b)
		return m, err
	case ClientMessageKindDisconnect:
		err = m.Disconnect.decode(b)
		return m, err
	case ClientMessageKindInput:
		err = m.Input.decode(b)
		return m, err
	case ClientMessageKindResize:
		err = m.Resize.decode(b)
		return m, err
	default:
		return ClientMessage{}, fmt.Errorf("unknown message kind: %v", m.Kind)
	}
}

func (m ClientMessage) Encode(b *bytes.Buffer) []byte {
	err := b.WriteByte(byte(m.Kind)) // err is always nil
	if err != nil {
		panic(err)
	}

	switch m.Kind {
	case ClientMessageKindInvalid:
		break
	case ClientMessageKindConnect:
		m.Connect.encode(b)
	case ClientMessageKindDisconnect:
		m.Disconnect.encode(b)
	case ClientMessageKindInput:
		m.Input.encode(b)
	case ClientMessageKindResize:
		m.Resize.encode(b)
	default:
		panic("non-exhaustive switch")
	}

	return b.Bytes()
}

func (m *ClientConnect) decode(b *bytes.Buffer) error {
	err := binary.Read(b, binary.LittleEndian, &m.StreamID)
	if err != nil {
		return err
	}
	m.ResourceName = b.String()
	return nil
}

func (m *ClientConnect) encode(b *bytes.Buffer) {
	err := binary.Write(b, binary.LittleEndian, m.StreamID) // Since bytes.Buffer Write never fails, this call should never fail either
	if err != nil {
		panic(err)
	}
	_, err = b.WriteString(m.ResourceName) // err is always nil
	if err != nil {
		panic(err)
	}
}

func (m *ClientDisconnect) decode(b *bytes.Buffer) error {
	return binary.Read(b, binary.LittleEndian, m)
}

func (m *ClientDisconnect) encode(b *bytes.Buffer) {
	err := binary.Write(b, binary.LittleEndian, m.StreamID) // Since bytes.Buffer Write never fails, this call should never fail either
	if err != nil {
		panic(err)
	}
}

func (m *ClientInput) decode(b *bytes.Buffer) error {
	err := binary.Read(b, binary.LittleEndian, &m.StreamID)
	if err != nil {
		return err
	}
	m.Data = b.Bytes()
	return nil
}

func (m *ClientInput) encode(b *bytes.Buffer) {
	err := binary.Write(b, binary.LittleEndian, m.StreamID) // Since bytes.Buffer Write never fails, this call should never fail either
	if err != nil {
		panic(err)
	}
	_, err = b.Write(m.Data) // err is always nil
	if err != nil {
		panic(err)
	}
}

func (m *ClientResize) decode(b *bytes.Buffer) error {
	return binary.Read(b, binary.LittleEndian, m)
}

func (m *ClientResize) encode(b *bytes.Buffer) {
	err := binary.Write(b, binary.LittleEndian, *m) // Since bytes.Buffer Write never fails, this call should never fail either
	if err != nil {
		panic(err)
	}
}
