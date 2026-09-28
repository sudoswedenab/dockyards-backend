package webshell

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type=ClientMessageKind
type ClientMessageKind uint8
const (
	ClientMessageKindInvalid        ClientMessageKind = 0
	ClientMessageKindAuthenticate   ClientMessageKind = 1
	ClientMessageKindInput          ClientMessageKind = 2
	ClientMessageKindResize         ClientMessageKind = 3
)

type ClientMessage struct {
	Authenticate struct {
		AccessToken string
	}

	Input struct {
		Data []byte
	}

	Resize struct {
		Rows uint16
		Cols uint16
		XPixel uint16 // width in pixels
		YPixel uint16 // height in pixels
	}

	Kind ClientMessageKind
}

func ParseClientMessage(raw []byte) (ClientMessage, error) {
	buf := bytes.NewBuffer(raw)
	var m ClientMessage
	err := binary.Read(buf, binary.LittleEndian, &m.Kind)
	if err != nil {
		return ClientMessage{}, fmt.Errorf("could not parse message kind: %w", err)
	}

	switch m.Kind {
	case ClientMessageKindInvalid:
		return m, nil
	case ClientMessageKindAuthenticate:
		m.Authenticate.AccessToken = buf.String()
	case ClientMessageKindInput:
		m.Input.Data = buf.Bytes()
	case ClientMessageKindResize:
		err := binary.Read(buf, binary.LittleEndian, &m.Resize)
		if err != nil {
			return ClientMessage{}, fmt.Errorf("could not decode resize message: %w", err)
		}
		return m, nil
	default:
		return ClientMessage{}, fmt.Errorf("unknown message kind: %v", m.Kind)
	}

	return m, nil
}

func ClientMessageAuthenticate(token string) ClientMessage {
	var m ClientMessage
	m.Kind = ClientMessageKindAuthenticate
	m.Authenticate.AccessToken = token
	return m
}

func ClientMessageInput(data []byte) ClientMessage {
	var m ClientMessage
	m.Kind = ClientMessageKindInput
	m.Input.Data = data
	return m
}

func ClientMessageResize(rows uint16, cols uint16, xpixel uint16, ypixel uint16) ClientMessage {
	var m ClientMessage
	m.Kind = ClientMessageKindResize
	m.Resize.Rows = rows
	m.Resize.Cols = cols
	m.Resize.XPixel = xpixel
	m.Resize.YPixel = ypixel
	return m
}

func (m ClientMessage) Encode(b *bytes.Buffer) []byte {
	err := b.WriteByte(byte(m.Kind)) // err is always nil
	if err != nil {
		panic(err)
	}

	switch m.Kind {
	case ClientMessageKindInvalid:
		break
	case ClientMessageKindAuthenticate:
		_, err  = b.WriteString(m.Authenticate.AccessToken) // err is always nil
		if err != nil {
			panic(err)
		}
	case ClientMessageKindInput:
		_, err = b.Write(m.Input.Data) // err is always nil
		if err != nil {
			panic(err)
		}
	case ClientMessageKindResize:
		err := binary.Write(b, binary.LittleEndian, m.Resize)
		if err != nil {
			panic(err) // Since bytes.Buffer Write never fails, this call should never fail either
		}
	default:
		panic("non-exhaustive switch")
	}

	return b.Bytes()
}

