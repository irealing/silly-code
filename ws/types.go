package ws

import (
	"errors"
)

var (
	closedSession        = errors.New("closed session")
	closedSessionManager = errors.New("closed session manager")
)

type localError struct {
	temporary bool
	timeout   bool
	describe  string
}

func newError(timeout, temporary bool, describe string) localError {
	return localError{temporary: temporary, timeout: timeout, describe: describe}
}
func (err localError) Timeout() bool {
	return err.timeout
}
func (err localError) Temporary() bool {
	return err.temporary
}
func (err localError) Error() string {
	return err.describe
}

type Message interface {
	Type() int
	Bytes() []byte
}
type ByteMessage struct {
	messageType int
	body        []byte
}

func NewByteMessage(t int, body []byte) Message {
	return &ByteMessage{t, body}
}
func (rm *ByteMessage) Type() int {
	return rm.messageType
}
func (rm *ByteMessage) Bytes() []byte {
	return rm.body
}
