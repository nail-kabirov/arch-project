package app

import (
	"arch-homework/pkg/common/app/uuid"
)

type RequestID uuid.UUID

type ProcessedRequestRepositoryRead interface {
	IsRequestProcessed(uid RequestID) (bool, error)
}

type ProcessedRequestRepository interface {
	ProcessedRequestRepositoryRead
	SetRequestProcessed(uid RequestID) (alreadyProcessed bool, err error)
}
