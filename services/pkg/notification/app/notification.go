package app

import (
	"arch-homework/pkg/common/app/uuid"
	"time"
)

type UserID uuid.UUID

type LotID uuid.UUID

type NotificationType string

const (
	TypeLotFinished NotificationType = "lotFinished"
	TypeLotClosed   NotificationType = "lotClosed"
	TypeLotWon      NotificationType = "lotWon"
	TypeLotSent     NotificationType = "lotSent"
	TypeLotReceived NotificationType = "lotReceived"
	TypeBidOutbid   NotificationType = "bidOutbid"
)

type Notification struct {
	Type         NotificationType
	UserID       UserID
	LotID        *LotID
	Message      string
	CreationDate time.Time
}

type NotificationRepository interface {
	Store(notification *Notification) error
	FindAllByUserID(userID UserID) ([]Notification, error)
}
