package app

import "arch-homework/pkg/common/app/uuid"

type UserID uuid.UUID

type LotInfo struct {
	ID         LotID
	OwnerID    UserID
	ReceiverID UserID
}

type LotServiceClient interface {
	FindFinishedLotInfo(id LotID) (*LotInfo, error)
}
