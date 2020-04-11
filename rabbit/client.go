package rabbit

import (
	"context"
	"sync"

	"github.com/op/go-logging"
	"github.com/streadway/amqp"
)

type mqClient struct {
	uri    string
	conn   *amqp.Connection
	logger *logging.Logger
	cLock  sync.Mutex
}

func NewClient(uri string) Client {
	return &mqClient{uri: uri, logger: logging.MustGetLogger("mqClient")}
}

func (mc *mqClient) Listen(ctx context.Context, opt *Options, callback MessageCallback) error {
	for {
		select {
		case <-ctx.Done():
			mc.logger.Notice("content done ,listen break.")
			return nil
		default:
			if err := mc.listen(ctx, opt, callback); err != nil {
				mc.logger.Warningf("failed to listen messages %s", err)
				continue
			}
		}
	}
}
func (mc *mqClient) reconnect() (_ *amqp.Connection, err error) {
	mc.cLock.Lock()
	defer mc.cLock.Unlock()
	if mc.conn == nil || mc.conn.IsClosed() {
		mc.conn, err = connectMQ(mc.uri, connectMQRetry)
	}
	return mc.conn, err
}
func (mc *mqClient) bind(opt *Options) (channel *amqp.Channel, err error) {
	conn, err := mc.reconnect()
	if err != nil {
		mc.logger.Warning("failed connection RabbitMQ after 3 retries")
		return
	}
	channel, err = conn.Channel()
	if err != nil {
		return
	}
	if err = channel.ExchangeDeclare(opt.Exchange, string(opt.Type), opt.Durable, false, false, false, nil); err != nil {
		mc.logger.Error("declare exchange error ", err)
		return
	}
	if _, err = channel.QueueDeclare(opt.Queue, opt.Durable, false, false, false, nil); err != nil {
		mc.logger.Error("declare queue error ", err)
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
		mc.logger.Warningf("ack failed %s", err)
	}
}

func (mc *mqClient) listen(ctx context.Context, opt *Options, callback MessageCallback) error {
	channel, err := mc.bind(opt)
	if err != nil {
		mc.logger.Errorf("bind %s error %s", opt, err)
		return err
	}
	delivery, err := channel.Consume(opt.Queue, "", opt.AutoAck, false, false, false, nil)
	if err != nil {
		mc.logger.Error("rabbitMQ consume error ", err)
		return err
	}
	defer func() {
		if err := channel.Close(); err != nil {
			mc.logger.Error("close channel error ", err)
		}
	}()
loop:
	for {
		select {
		case <-ctx.Done():
			mc.logger.Notice("receive context done signal break listen")
			break loop
		case msg, ok := <-delivery:
			if !ok {
				mc.logger.Notice("delivery closed ,break listen")
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
	return &publisher{reconnect: mc.reconnect, opt: opt}
}
func (mc *mqClient) Close() {
	mc.cLock.Lock()
	defer mc.cLock.Unlock()
	if mc.conn == nil || mc.conn.IsClosed() {
		mc.logger.Critical("rabbit.Client already closed.")
		mc.conn = nil
	}
	if err := mc.conn.Close(); err != nil {
		mc.logger.Critical("rabbit.Client close error", err)
	}
}
