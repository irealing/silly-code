package rabbit

import (
	"github.com/streadway/amqp"
)

func connectMQ(uri string, retry int) (*amqp.Connection, error) {
	var err error
	var conn *amqp.Connection
	for i := 0; i < retry+1; i++ {
		logger.Infof("the %dth attempt to connect to rabbitMQ ", i)
		conn, err = amqp.Dial(uri)
		if err != nil {
			logger.Warningf("the %dth connection to rabbitMQ failed %s", i, err.Error())
			continue
		} else {
			break
		}
	}
	return conn, err
}
