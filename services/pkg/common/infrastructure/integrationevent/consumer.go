package integrationevent

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/streams"
	"arch-homework/pkg/common/app/uuid"

	"encoding/json"

	"github.com/cenkalti/backoff/v4"
	"github.com/rabbitmq/rabbitmq-stream-go-client/pkg/amqp"
	"github.com/sirupsen/logrus"
)

func StartEventConsumer(rmqEnv streams.Environment, handler integrationevent.EventHandler, logger *logrus.Logger) error {
	consumer, err := rmqEnv.AddConsumer(streams.IntegrationEventStreamName)
	if err != nil {
		return err
	}
	eventConsumer := &eventConsumer{
		consumer: consumer,
		handler:  handler,
		logger:   logger,
	}
	consumer.SetMessageHandler(eventConsumer.messageHandler)

	return nil
}

type eventConsumer struct {
	consumer streams.Consumer
	handler  integrationevent.EventHandler
	logger   *logrus.Logger
}

func (ec *eventConsumer) messageHandler(msg *amqp.Message) {
	data := msg.Data
	if len(data) == 0 {
		ec.logger.Error("received message without body")
		return
	}
	if len(data) > 1 {
		ec.logger.Warnf("received data with multiple data - %v", data)
	}
	rawData := data[0]

	var eventData EventDataView
	err := json.Unmarshal(rawData, &eventData)
	if err != nil {
		ec.logger.Error("unsupported message body")
		return
	}
	err = uuid.ValidateUUID(eventData.UID)
	if err != nil {
		ec.logger.Error("invalid event uid")
		return
	}

	ec.handleEvent(integrationevent.EventData{
		UID:  integrationevent.EventUID(eventData.UID),
		Type: eventData.Type,
		Body: eventData.Body,
	})
}

func (ec *eventConsumer) handleEvent(eventData integrationevent.EventData) {
	err := backoff.Retry(func() error {
		err2 := ec.handler.Handle(eventData)
		if err2 != nil {
			ec.logger.Errorf("error processing integration event - '%s'. attempt to retry", err2.Error())
		}
		return err2
	}, backoff.NewExponentialBackOff())

	if err != nil {
		ec.logger.Fatalf("error processing integration event - %s\nDetails: uid - '%s', type - '%s', body - '%s'", err.Error(), string(eventData.UID), eventData.Type, eventData.Body)
	} else {
		ec.logger.Infof("integration event '%s' with type '%s' handled", string(eventData.UID), eventData.Type)
	}
}
