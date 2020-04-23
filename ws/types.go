package ws

import (
	"errors"

	"github.com/op/go-logging"
)

var (
	closedSession        = errors.New("closed session")
	closedSessionManager = errors.New("closed session manager")
	wsLogger             = logging.MustGetLogger("WS")
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
type RawMessage struct {
	messageType int
	body        []byte
}

func NewRawMessage(t int, body []byte) Message {
	return &RawMessage{t, body}
}
func (rm *RawMessage) Type() int {
	return rm.messageType
}
func (rm *RawMessage) Bytes() []byte {
	return rm.body
}
