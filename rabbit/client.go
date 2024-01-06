package rabbit

import (
	"context"
	sillyKits "github.com/irealing/silly-kits"
	amqp "github.com/rabbitmq/amqp091-go"
	"log/slog"
	"sync"
	"time"
)

type mqClient struct {
	uri    string
	conn   *amqp.Connection
	logger *slog.Logger
	cLock  sync.Mutex
}

func NewClient(uri string, logger *slog.Logger) Client {
	return &mqClient{uri: uri, logger: logger}
}

func (mc *mqClient) Listen(ctx context.Context, opt *Options, callback MessageCallback) error {
	for {
		select {
		case <-ctx.Done():
			mc.logger.Debug("content done ,listen break.")
			return nil
		default:
			if err := mc.listen(ctx, opt, callback); err != nil {
				mc.logger.Warn("failed to listen messages", "error", err)
				continue
			}
		}
	}
}
func (mc *mqClient) reconnect(retry int, delay time.Duration) (_ *amqp.Connection, err error) {
	mc.cLock.Lock()
	defer mc.cLock.Unlock()
	if mc.conn == nil || mc.conn.IsClosed() {
		mc.conn, err = sillyKits.Retry(func() (*amqp.Connection, error) {
			return connectMQ(mc.uri, connectMQRetry)
		}, retry, delay)
	}
	return mc.conn, err
}
func (mc *mqClient) bind(opt *Options) (channel *amqp.Channel, err error) {
	conn, err := mc.reconnect(3, time.Second)
	if err != nil {
		mc.logger.Warn("failed connection RabbitMQ after retry")
		return
	}
	channel, err = conn.Channel()
	if err != nil {
		return
	}
	if err = channel.ExchangeDeclare(opt.Exchange, string(opt.Type), opt.Durable, false, false, false, nil); err != nil {
		mc.logger.Error("declare exchange ", "error", err)
		return
	}
	if _, err = channel.QueueDeclare(opt.Queue, opt.Durable, false, false, false, nil); err != nil {
		mc.logger.Error("declare queue error ", "error", err)
		return
	}
	return channel, channel.QueueBind(opt.Queue, opt.RoutingKey, opt.Exchange, false, nil)
}

func (mc *mqClient) safeAck(msg *amqp.Delivery, success bool) {
	var err error
	if success {
		err = msg.Ack(true)
	} else {
		err = msg.Nack(false, true)
	}
	if err != nil {
		mc.logger.Warn("ack failed", "error", err)
	}
}

func (mc *mqClient) listen(ctx context.Context, opt *Options, callback MessageCallback) error {
	channel, err := mc.bind(opt)
	if err != nil {
		mc.logger.Warn("bind error", "options", opt, "error", err)
		return err
	}
	delivery, err := channel.Consume(opt.Queue, "", opt.AutoAck, false, false, false, nil)
	if err != nil {
		mc.logger.Warn("rabbitMQ consume error", "error", err)
		return err
	}
	defer func() {
		if err := channel.Close(); err != nil {
			mc.logger.Warn("close channel error ", "error", err)
		}
	}()
loop:
	for {
		select {
		case <-ctx.Done():
			mc.logger.Debug("receive context done signal break listen")
			break loop
		case msg, ok := <-delivery:
			if !ok {
				mc.logger.Debug("delivery closed ,break listen")
				break loop
			}
			err := callback(msg.Body)
			if !opt.AutoAck {
				mc.safeAck(&msg, err == nil)
			}
		}
	}
	return nil
}
func (mc *mqClient) Publisher(opt *Options) Publisher {
	return &publisher{reconnect: func() (*amqp.Connection, error) {
		return mc.reconnect(connectMQRetry, time.Second)
	}, opt: opt, logger: mc.logger}
}
func (mc *mqClient) Close() {
	mc.cLock.Lock()
	defer mc.cLock.Unlock()
	if mc.conn == nil || mc.conn.IsClosed() {
		mc.logger.Warn("rabbit.Client already closed.")
		mc.conn = nil
	}
	if err := mc.conn.Close(); err != nil {
		mc.logger.Warn("rabbit.Client close error", "error", err)
	}
}
