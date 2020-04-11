package rabbit

import (
	"context"
	"fmt"
	"github.com/op/go-logging"
)

type ExchangeType string

const (
	Fanout         ExchangeType = "fanout"
	Direct                      = "direct"
	Topic                       = "topic"
	connectMQRetry              = 3
)

var logger = logging.MustGetLogger("rabbit")

type MessageCallback func([]byte) error

type Encoder interface {
	MimeType() string
	Encode(interface{}) (string, []byte, error)
}

type Options struct {
	Type       ExchangeType
	Exchange   string
	Queue      string
	RoutingKey string
	AutoAck    bool
	Durable    bool
}

func (opt *Options) String() string {
	return fmt.Sprintf("Options(Type: %s; Exchange: %s;Queue: %s;RoutingKey: %s)", opt.Type, opt.Exchange, opt.Queue, opt.RoutingKey)
}

type Publisher interface {
	Publishing(key, mineType string, data []byte) error
	PublishJson(string, interface{}) error
	Publish(encoder Encoder, data interface{}) error
}

type Client interface {
	Listen(ctx context.Context, opt *Options, callback MessageCallback) error
	Publisher(opt *Options) Publisher
}
