package app

import (
	"arch-homework/pkg/common/app/uuid"
)

type UserID uuid.UUID
type LotID uuid.UUID
type AccountEventType string

const (
	createAccountEventType  AccountEventType = "create_account"
	topUpAccountEventType   AccountEventType = "top_up_account"
	blockPaymentEventType   AccountEventType = "block_payment"
	unblockPaymentEventType AccountEventType = "unblock_payment"
	finishPaymentEventType  AccountEventType = "finish_payment"
	receivePaymentEventType AccountEventType = "receive_payment"
)

type UserAccountEvent struct {
	UserID    UserID
	LotID     *LotID
	EventType AccountEventType
	Amount    Amount
}

type UserAccountEventRepositoryRead interface {
	FindAllByUserID(id UserID) ([]UserAccountEvent, error)
}

type UserAccountEventRepository interface {
	UserAccountEventRepositoryRead
	Store(event *UserAccountEvent) error
}
