package rabbit

import (
	"encoding/json"
	"github.com/op/go-logging"
	"github.com/streadway/amqp"
)

type publisher struct {
	reconnect func() (*amqp.Connection, error)
	conn      *amqp.Connection
	ch        *amqp.Channel
	opt       *Options
	logger    *logging.Logger
}

func (p *publisher) init() (Publisher, error) {
	var err error
	if p.conn != nil && !p.conn.IsClosed() {
		return p, err
	}
	p.conn, err = p.reconnect()
	if err != nil {
		return nil, err
	}
	p.ch, err = p.conn.Channel()
	return p, p.ch.ExchangeDeclare(p.opt.Exchange, string(p.opt.Type), p.opt.Durable, false, false, false, nil)
}

func (p *publisher) Publishing(key, mineType string, data []byte) error {
	if _, err := p.init(); err != nil {
		return err
	}
	if key == "" {
		key = p.opt.RoutingKey
	}
	return p.ch.Publish(p.opt.Exchange, key, false, false, amqp.Publishing{Type: mineType, Body: data})
}

func (p *publisher) Publish(encoder Encoder, data interface{}) error {
	routing, ret, err := encoder.Encode(data)
	if err != nil {
		p.logger.Warning("encode data error", err)
		return err
	}
	return p.Publishing(routing, encoder.MimeType(), ret)
}

func (p *publisher) PublishJson(key string, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		p.logger.Warning("serialize data error", err)
		return err
	}
	return p.Publishing(key, "application/json", data)
}
