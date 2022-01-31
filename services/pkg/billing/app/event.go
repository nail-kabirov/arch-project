package app

import "arch-homework/pkg/common/app/integrationevent"

type ProcessedEventRepository interface {
	SetEventProcessed(uid integrationevent.EventUID) (alreadyProcessed bool, err error)
}

type UserEvent interface {
	UserID() UserID
}

func NewUserRegisteredEvent(userID UserID, login string) UserEvent {
	return userRegisteredEvent{userID: userID, login: login}
}

func NewLotBidOutbidEvent(userID UserID, lotID LotID, bidAmount Amount) UserEvent {
	return lotBidOutbidEvent{
		userID:    userID,
		lotID:     lotID,
		bidAmount: bidAmount,
	}
}
func NewLotBidCancelledEvent(userID UserID, lotID LotID, bidAmount Amount) UserEvent {
	return lotBidCancelledEvent{
		userID:    userID,
		lotID:     lotID,
		bidAmount: bidAmount,
	}
}

func NewLotReceivedEvent(userID UserID, lotID LotID, lotOwnerID UserID, finalAmount Amount) UserEvent {
	return lotReceivedEvent{
		userID:      userID,
		lotID:       lotID,
		lotOwnerID:  lotOwnerID,
		finalAmount: finalAmount,
	}
}

type userRegisteredEvent struct {
	userID UserID
	login  string
}

func (e userRegisteredEvent) UserID() UserID {
	return e.userID
}

type lotBidOutbidEvent struct {
	userID    UserID
	lotID     LotID
	bidAmount Amount
}

func (e lotBidOutbidEvent) UserID() UserID {
	return e.userID
}

type lotBidCancelledEvent struct {
	userID    UserID
	lotID     LotID
	bidAmount Amount
}

func (e lotBidCancelledEvent) UserID() UserID {
	return e.userID
}

type lotReceivedEvent struct {
	userID      UserID
	lotID       LotID
	lotOwnerID  UserID
	finalAmount Amount
}

func (e lotReceivedEvent) UserID() UserID {
	return e.userID
}
