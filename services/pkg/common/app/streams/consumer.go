package streams

import "github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"

type MessageHandlerCallback func(msg *amqp.Message)

type Consumer interface {
	SetMessageHandler(cb MessageHandlerCallback)
}
