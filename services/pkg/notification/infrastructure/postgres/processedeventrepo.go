package postgres

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/notification/app"

	"database/sql"

	"github.com/pkg/errors"
)

func NewProcessedEventRepository(client postgres.Client) app.ProcessedEventRepository {
	return &processedEventRepository{client: client}
}

type processedEventRepository struct {
	client postgres.Client
}

func (repo *processedEventRepository) SetEventProcessed(uid integrationevent.EventUID) (alreadyProcessed bool, err error) {
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
