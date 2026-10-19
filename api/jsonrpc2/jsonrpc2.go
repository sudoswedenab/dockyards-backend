// Copyright 2026 Sudo Sweden AB
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jsonrpc2

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

type JSONRPC struct {}

func (JSONRPC) MarshalJSON() ([]byte, error) {
	return []byte("\"2.0\""), nil
}

func (*JSONRPC) UnmarshalJSON(data []byte) error {
	var value string
	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	if value != "2.0" {
		return errors.New("expected jsonrpc version to be 2.0")
	}

	return nil
}

type Sequence struct {
	sequence int
}

func (s *Sequence) Next() ID {
	if s.sequence < 0 {
		s.sequence = 0
	}
	if s.sequence >= math.MaxInt {
		s.sequence = 0
	}
	s.sequence++
	return ID{kind: IDKindNumber, intValue: s.sequence}
}

type IDKind int
const (
	IDKindNone   IDKind = iota
	IDKindNumber
	IDKindString
)

type ID struct {
	kind        IDKind
	intValue    int
	stringValue string
}

func (id *ID) IsZero() bool {
	return id.kind == IDKindNone
}

func (id ID) MarshalJSON() ([]byte, error) {
	switch id.kind {
	case IDKindNone: return nil, nil
	case IDKindNumber: return json.Marshal(id.intValue)
	case IDKindString: return json.Marshal(id.stringValue)
	}
	return nil, errors.New("id was nether int nor string")
}

func (id *ID) UnmarshalJSON(data []byte) error {
	if json.Unmarshal(data, &id.intValue) == nil {
		id.kind = IDKindNumber
		return nil
	}

	if json.Unmarshal(data, &id.stringValue) == nil {
		id.kind = IDKindString
		return nil
	}

	return errors.New("id must be either string or int")
}

func (id *ID) String() string {
	switch id.kind {
	case IDKindNumber: return fmt.Sprint(id.intValue)
	case IDKindString: return id.stringValue
	}
	return ""
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func (e *Error) IsZero() bool {
	return e.Code == 0
}

var ErrParse          = Error{-32700, "Invalid JSON was received by the server", nil}
var ErrInvalidRequest = Error{-32600, "The JSON sent is not a valid Request object.", nil}
var ErrMethodNotFound = Error{-32601, "The method does not exist / is not available.", nil}
var ErrInvalidParams  = Error{-32602, "Invalid method parameter(s).", nil}
var ErrInternal       = Error{-32603, "Internal JSON-RPC error.", nil}

var ErrUnauthorized   = ErrorImplementationDefined(-32001, "Unauthorized")

// Code must be between -32099 <= code <= -32000
func ErrorImplementationDefined(code int, message string) Error {
	if code >= -32000 || code <= -32099 {
		code = ErrInternal.Code
	}
	return Error{code, message, nil}
}

type Message struct {
	JSONRPC JSONRPC         `json:"jsonrpc"` // MUST be exactly "2.0".
	ID      ID              `json:"id,omitzero"`

	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitzero"`

	Error   Error           `json:"error,omitzero"`
	Result  json.RawMessage `json:"result,omitzero"`
}

type MessageKind int
const (
	MessageKindNotification MessageKind = 0b00
	MessageKindRequest      MessageKind = 0b01

	MessageKindError        MessageKind = 0b10
	MessageKindResult       MessageKind = 0b11
)

func (m MessageKind) IsRequestLike() bool {
	return (m & 0b10) == 0b00
}

func (m MessageKind) IsResponseLike() bool {
	return (m & 0b10) == 0b10
}

func (m *Message) Kind() MessageKind {
	if m.Method != "" {
		if m.ID.IsZero() {
			return MessageKindNotification
		}
		return MessageKindRequest
	}
	if !m.Error.IsZero() {
		return MessageKindError
	}
	return MessageKindResult
}

func NewRequestWithBody(method string, id ID, params any) (Message, error) {
	req := NewRequest(method, id)

	var err error
	req.Params, err = json.Marshal(params)

	return req, err
}

func NewRequest(method string, id ID) Message {
	return Message{
		Method: method,
		ID: id,
	}
}

func NewNotificationWithBody(method string, params any) (Message, error) {
	req := NewNotification(method)
	var err error
	req.Params, err = json.Marshal(params)
	return req, err
}

func NewNotification(method string) Message {
	return Message{
		Method: method,
	}
}

func NewResult(id ID) Message {
	return NewResultWithMessage(id, json.RawMessage("null"))
}

func NewResultWithMessage(id ID, result json.RawMessage) Message {
	return Message{
		ID: id,
		Result: result,
	}
}

func NewResultWithBody(id ID, result any) (Message, error) {
	body, err := json.Marshal(result)
	if err != nil {
		return Message{}, err
	}
	return NewResultWithMessage(id, body), nil
}

func NewError(id ID, err Error) Message {
	return Message{
		ID: id,
		Error: err,
	}
}
