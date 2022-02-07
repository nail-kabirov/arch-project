package app

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/storedevent"
)

type IntegrationEventParser interface {
	ParseIntegrationEvent(event integrationevent.EventData) (HandledEvent, error)
}

type ProcessedEventRepository interface {
	SetEventProcessed(uid integrationevent.EventUID) (alreadyProcessed bool, err error)
}

func NewEventHandler(
	dbDependency DBDependency,
	parser IntegrationEventParser,
	eventSender storedevent.Sender,
	billingClient BillingClient,
) integrationevent.EventHandler {
	return &eventHandler{
		readRepoProvider: dbDependency,
		trUnitFactory:    dbDependency,
		parser:           parser,
		eventSender:      eventSender,
		billingClient:    billingClient,
	}
}

type eventHandler struct {
	readRepoProvider ReadRepositoryProvider
	trUnitFactory    TransactionalUnitFactory
	parser           IntegrationEventParser
	eventSender      storedevent.Sender
	billingClient    BillingClient
}

func (handler *eventHandler) Handle(event integrationevent.EventData) error {
	parsedEvent, err := handler.parser.ParseIntegrationEvent(event)
	if err != nil || parsedEvent == nil {
		return err
	}

	return handler.executeInTransaction(func(trUnit TransactionalUnit) error {
		requestRepo := trUnit.ProcessedEventRepository()
		alreadyProcessed, err := requestRepo.SetEventProcessed(event.UID)
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		service := NewLotService(handler.readRepoProvider, trUnit, handler.eventSender, handler.billingClient)

		switch e := parsedEvent.(type) {
		case deliveryLotSentEvent:
			return service.SetLotSent(e.lotID)
		case deliveryLotReceivedEvent:
			return service.SetLotReceived(e.lotID)
		default:
			return nil
		}
	})
}

func (handler *eventHandler) executeInTransaction(f func(TransactionalUnit) error) (err error) {
	var trUnit TransactionalUnit
	trUnit, err = handler.trUnitFactory.NewTransactionalUnit()
	if err != nil {
		return err
	}
	defer func() {
		err = trUnit.Complete(err)
	}()
	err = f(trUnit)
	return err
}
