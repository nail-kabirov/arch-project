package app

import (
	"arch-homework/pkg/common/app/uuid"
)

type RequestID uuid.UUID

type ProcessedRequestRepository interface {
	SetRequestProcessed(uid RequestID) (alreadyProcessed bool, err error)
}
