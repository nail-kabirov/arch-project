package app

import (
	"arch-homework/pkg/common/app/uuid"
	"time"
)

type UserID uuid.UUID

type Bid struct {
	LotID        LotID
	UserID       UserID
	Amount       Amount
	CreationTime time.Time
}

type BidRepository interface {
	TryFindLastByLotID(lotID LotID) (*Bid, error)
	Store(bid *Bid) error
}
