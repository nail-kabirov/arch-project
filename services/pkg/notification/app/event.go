package app

import (
	"arch-homework/pkg/common/app/integrationevent"
)

type ProcessedEventRepository interface {
	SetEventProcessed(uid integrationevent.EventUID) (alreadyProcessed bool, err error)
}

type HandledEvent interface {
}

func NewLotWonEvent(lotID LotID, lotOwnerID, userID UserID) HandledEvent {
	return lotWonEvent{
		lotID:      lotID,
		lotOwnerID: lotOwnerID,
		userID:     userID,
	}
}

func NewLotSentEvent(lotID LotID, lotOwnerID, userID UserID) HandledEvent {
	return lotSentEvent{
		lotID:      lotID,
		lotOwnerID: lotOwnerID,
		userID:     userID,
	}
}

func NewLotReceivedEvent(lotID LotID, lotOwnerID, userID UserID) HandledEvent {
	return lotReceivedEvent{
		lotID:      lotID,
		lotOwnerID: lotOwnerID,
		userID:     userID,
	}
}

func NewLotClosedEvent(lotID LotID, lotOwnerID UserID) HandledEvent {
	return lotClosedEvent{
		lotID:      lotID,
		lotOwnerID: lotOwnerID,
	}
}

func NewBidOutbidEvent(lotID LotID, userID UserID) HandledEvent {
	return bidOutbidEvent{
		lotID:  lotID,
		userID: userID,
	}
}

type lotWonEvent struct {
	lotID      LotID
	lotOwnerID UserID
	userID     UserID
}

type lotSentEvent struct {
	lotID      LotID
	lotOwnerID UserID
	userID     UserID
}

type lotReceivedEvent struct {
	lotID      LotID
	lotOwnerID UserID
	userID     UserID
}

type lotClosedEvent struct {
	lotID      LotID
	lotOwnerID UserID
}

type bidOutbidEvent struct {
	lotID  LotID
	userID UserID
}
