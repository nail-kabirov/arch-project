package streams

import (
	"arch-homework/pkg/common/app/streams"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func newConsumer(consumer *stream.Consumer) consumer {
	return &streamConsumer{consumer: consumer}
}

type consumer interface {
	streams.Consumer
	GetMessageHandler() streams.MessageHandlerCallback
}

type streamConsumer struct {
	consumer *stream.Consumer
	handler  streams.MessageHandlerCallback
}

func (s *streamConsumer) GetMessageHandler() streams.MessageHandlerCallback {
	return s.handler
}

func (s *streamConsumer) SetMessageHandler(cb streams.MessageHandlerCallback) {
	s.handler = cb
}
