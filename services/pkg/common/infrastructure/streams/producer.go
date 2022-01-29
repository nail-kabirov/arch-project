package streams

import (
	"arch-homework/pkg/common/app/streams"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/message"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

func newProducer(producer *stream.Producer) streams.Producer {
	return &streamProducer{producer: producer}
}

type streamProducer struct {
	producer *stream.Producer
}

func (s *streamProducer) Send(msg streams.Msg) error {
	return s.producer.Send(toStreamMsg(msg))
}

func (s *streamProducer) BatchSend(messages []streams.Msg) error {
	streamMessages := make([]message.StreamMessage, 0, len(messages))
	for _, msg := range messages {
		streamMessages = append(streamMessages, toStreamMsg(msg))
	}
	return s.producer.BatchSend(streamMessages)
}

func toStreamMsg(msg streams.Msg) message.StreamMessage {
	streamMsg := amqp.NewMessage([]byte(msg.Body))
	streamMsg.SetPublishingId(int64(msg.ID))
	return streamMsg
}
