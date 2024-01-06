package rabbit

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func connectMQ(uri string, retry int) (*amqp.Connection, error) {
	var err error
	var conn *amqp.Connection
	for i := 0; i < retry+1; i++ {
		conn, err = amqp.Dial(uri)
		if err != nil {
			continue
		} else {
			break
		}
	}
	return conn, err
}
