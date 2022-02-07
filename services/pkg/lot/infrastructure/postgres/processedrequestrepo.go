package postgres

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/lot/app"

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

func (repo *processedRequestRepository) IsRequestProcessed(uid app.RequestID) (bool, error) {
	const query = `SELECT uid from processed_request WHERE uid = $1`

	var resUID string
	err := repo.client.Get(&resUID, query, string(uid))
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func (repo *processedRequestRepository) SetRequestProcessed(uid app.RequestID) (alreadyProcessed bool, err error) {
	return repo.setRequestProcessed(string(uid))
}

func (repo *processedRequestRepository) SetEventProcessed(uid integrationevent.EventUID) (alreadyProcessed bool, err error) {
	return repo.setRequestProcessed(string(uid))
}

func (repo *processedRequestRepository) setRequestProcessed(uid string) (bool, error) {
	const query = `INSERT INTO processed_request (uid) VALUES ($1) ON CONFLICT DO NOTHING RETURNING uid`

	var resUID string
	err := repo.client.Get(&resUID, query, uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, errors.WithStack(err)
	}
	return false, nil
}
