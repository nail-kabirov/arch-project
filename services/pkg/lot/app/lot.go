package app

import (
	"arch-homework/pkg/common/app/uuid"
	"errors"
	"time"
)

var ErrLotNotFound = errors.New("lot not found")

type LotID uuid.UUID
type LotStatus string

const (
	LotStatusActive   LotStatus = "active"
	LotStatusClosed   LotStatus = "closed"
	LotStatusFinished LotStatus = "finished"
	LotStatusSent     LotStatus = "sent"
	LotStatusReceived LotStatus = "received"
)

type Lot struct {
	ID            LotID
	OwnerID       UserID
	Description   string
	StartPrice    Amount
	BuyItNowPrice *Amount
	Status        LotStatus
	EndTime       time.Time
	CreationTime  time.Time
}

type LotSpecification struct {
	SearchString        *string
	ParticipationUserID *UserID
	WinnerUserID        *UserID
	OwnerUserID         *UserID
}

type LotRepositoryRead interface {
	FindByID(id LotID) (*Lot, error)
	FindActiveCompletedLots() ([]Lot, error)
}

type LotRepository interface {
	LotRepositoryRead
	Store(lot *Lot) error
}
