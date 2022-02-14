package app

import (
	"arch-homework/pkg/common/app/integrationevent"
)

type IntegrationEventParser interface {
	ParseIntegrationEvent(event integrationevent.EventData) (HandledEvent, error)
}

func NewEventHandler(trUnitFactory TransactionalUnitFactory, parser IntegrationEventParser) integrationevent.EventHandler {
	return &eventHandler{
		trUnitFactory: trUnitFactory,
		parser:        parser,
	}
}

type eventHandler struct {
	trUnitFactory TransactionalUnitFactory
	parser        IntegrationEventParser
}

func (handler *eventHandler) Handle(event integrationevent.EventData) error {
	parsedEvent, err := handler.parser.ParseIntegrationEvent(event)
	if err != nil || parsedEvent == nil {
		return err
	}

	return handler.executeInTransaction(func(provider RepositoryProvider) error {
		eventRepo := provider.ProcessedEventRepo()
		alreadyProcessed, err := eventRepo.SetEventProcessed(event.UID)
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return nil
		}

		service := NewNotificationService(provider.NotificationRepository())

		switch e := parsedEvent.(type) {
		case lotWonEvent:
			return handleLotWonEvent(service, e)
		case lotClosedEvent:
			return handleLotClosedEvent(service, e)
		case lotSentEvent:
			return handleLotSentEvent(service, e)
		case lotReceivedEvent:
			return handleLotReceivedEvent(service, e)
		case bidOutbidEvent:
			return handleBidOutbidEvent(service, e)
		default:
			return nil
		}
	})
}

func (handler *eventHandler) executeInTransaction(f func(RepositoryProvider) error) (err error) {
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

func handleLotWonEvent(service NotificationService, e lotWonEvent) error {
	err := service.AddNotification(TypeLotFinished, e.lotID, e.lotOwnerID)
	if err != nil {
		return err
	}
	return service.AddNotification(TypeLotWon, e.lotID, e.userID)
}

func handleLotClosedEvent(service NotificationService, e lotClosedEvent) error {
	return service.AddNotification(TypeLotClosed, e.lotID, e.lotOwnerID)
}

func handleLotSentEvent(service NotificationService, e lotSentEvent) error {
	return service.AddNotification(TypeLotSent, e.lotID, e.userID)
}

func handleLotReceivedEvent(service NotificationService, e lotReceivedEvent) error {
	return service.AddNotification(TypeLotReceived, e.lotID, e.lotOwnerID)
}

func handleBidOutbidEvent(service NotificationService, e bidOutbidEvent) error {
	return service.AddNotification(TypeBidOutbid, e.lotID, e.userID)
}
