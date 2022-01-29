package streams

import (
	"arch-homework/pkg/common/app/streams"

	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/stream"
)

const maxStreamAge = time.Hour * 24

func NewEnvironment(serviceName string, cfg streams.Config) (streams.Environment, error) {
	port, err := strconv.Atoi(cfg.Port)
	if err != nil {
		return nil, err
	}
	env, err := stream.NewEnvironment(
		stream.NewEnvironmentOptions().
			SetHost(cfg.Host).
			SetPort(port).
			SetUser(cfg.User).
			SetPassword(cfg.Password),
	)
	if err != nil {
		return nil, err
	}
	return &environment{
		serviceName: serviceName,
		env:         env,
		consumerMap: make(map[string]consumer),
	}, nil
}

type environment struct {
	serviceName string
	env         *stream.Environment
	consumerMap map[string]consumer
}

func (e *environment) AddConsumer(streamName string) (streams.Consumer, error) {
	if err := e.declareStream(streamName); err != nil {
		return nil, err
	}

	consumerName := e.consumerName(streamName)
	if _, ok := e.consumerMap[consumerName]; ok {
		return nil, errors.New("consumer for this stream already created")
	}

	options := stream.NewConsumerOptions().
		SetConsumerName(consumerName).
		SetCRCCheck(true).
		SetOffset(stream.OffsetSpecification{}.LastConsumed()).
		SetAutoCommit(stream.NewAutoCommitStrategy().SetCountBeforeStorage(2))

	handleMessages := func(consumerContext stream.ConsumerContext, message *amqp.Message) {
		name := consumerContext.Consumer.GetName()
		consumer, ok := e.consumerMap[name]
		if ok {
			if handler := consumer.GetMessageHandler(); handler != nil {
				handler(message)
			}
		}
	}
	rawConsumer, err := e.env.NewConsumer(streamName, handleMessages, options)
	if err != nil {
		return nil, err
	}
	consumer := newConsumer(rawConsumer)
	e.consumerMap[consumerName] = consumer

	return consumer, nil
}

func (e *environment) AddProducer(streamName string, cb streams.ConfirmationCallback) (streams.Producer, error) {
	if err := e.declareStream(streamName); err != nil {
		return nil, err
	}
	options := stream.NewProducerOptions().SetProducerName(e.producerName(streamName))
	producer, err := e.env.NewProducer(streamName, options)
	if err != nil {
		return nil, err
	}
	handlePublishConfirm(producer.NotifyPublishConfirmation(), cb)

	return newProducer(producer), nil
}

func (e *environment) declareStream(streamName string) error {
	exists, err := e.env.StreamExists(streamName)
	if exists || err != nil {
		return err
	}
	err = e.env.DeclareStream(streamName, stream.NewStreamOptions().SetMaxAge(maxStreamAge))
	return err
}

func (e *environment) producerName(streamName string) string {
	return fmt.Sprintf("%s_%s_producer", e.serviceName, streamName)
}

func (e *environment) consumerName(streamName string) string {
	return fmt.Sprintf("%s_%s_consumer", e.serviceName, streamName)
}

func handlePublishConfirm(confirms stream.ChannelPublishConfirm, cb streams.ConfirmationCallback) {
	go func() {
		for confirmed := range confirms {
			for _, msg := range confirmed {
				cb(streams.MsgID(msg.GetPublishingId()), msg.IsConfirmed(), msg.GetError())
			}
		}
	}()
}
