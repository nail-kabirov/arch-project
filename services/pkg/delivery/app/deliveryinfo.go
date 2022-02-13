package app

import (
	"arch-homework/pkg/common/app/uuid"

	"errors"
)

var ErrLotNotFound = errors.New("lot not found")

type LotID uuid.UUID
type LotStatus string
type TrackingID string
type Address string

const (
	LotStatusFinished LotStatus = "finished"
	LotStatusSent     LotStatus = "sent"
	LotStatusReceived LotStatus = "received"
)

type DeliveryInfo struct {
	LotID             LotID
	LotStatus         LotStatus
	TrackingID        *TrackingID
	ReceiverID        UserID
	ReceiverLogin     string
	ReceiverFirstName string
	ReceiverLastName  string
	ReceiverAddress   Address
	SenderID          UserID
	SenderLogin       string
	SenderFirstName   string
	SenderLastName    string
}

type DeliveryInfoRepositoryRead interface {
	FindByLotID(id LotID) (*DeliveryInfo, error)
}

type DeliveryInfoRepository interface {
	DeliveryInfoRepositoryRead
	Store(info *DeliveryInfo) error
}
