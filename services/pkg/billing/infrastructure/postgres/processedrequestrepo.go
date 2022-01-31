package postgres

import (
	"arch-homework/pkg/billing/app"
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/infrastructure/postgres"

	"database/sql"

	"github.com/pkg/errors"
)

func NewProcessedRequestRepository(client postgres.Client) app.ProcessedRequestRepository {
	return &processedRequestRepository{client: client}
}

func NewProcessedEventRepository(client postgres.Client) app.ProcessedEventRepository {
	return &processedRequestRepository{client: client}
}

type processedRequestRepository struct {
	client postgres.Client
}

func (repo *processedRequestRepository) SetEventProcessed(uid integrationevent.EventUID) (alreadyProcessed bool, err error) {
	return repo.setProcessed(uuid.UUID(uid))
}

func (repo *processedRequestRepository) SetRequestProcessed(uid app.RequestID) (alreadyProcessed bool, err error) {
	return repo.setProcessed(uuid.UUID(uid))
}

func (repo *processedRequestRepository) setProcessed(uid uuid.UUID) (alreadyProcessed bool, err error) {
	const query = `INSERT INTO processed_request (uid) VALUES ($1) ON CONFLICT DO NOTHING RETURNING uid`

	var resUID string
	err = repo.client.Get(&resUID, query, string(uid))
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, errors.WithStack(err)
	}
	return false, nil
}
