package postgres

import (
	"arch-homework/pkg/billing/app"
	"arch-homework/pkg/common/infrastructure/postgres"
	"database/sql"

	"github.com/pkg/errors"
)

func NewUserAccountEventRepository(client postgres.Client) app.UserAccountEventRepository {
	return &userAccountEventRepository{client: client}
}

type userAccountEventRepository struct {
	client postgres.Client
}

func (repo *userAccountEventRepository) Store(event *app.UserAccountEvent) error {
	const query = `
			INSERT INTO user_account_event (user_id, lot_id, event_type, amount)
			VALUES (:user_id, :lot_id, :event_type, :amount)
		`

	accountEvent := sqlxUserAccountEvent{
		UserID:    string(event.UserID),
		EventType: string(event.EventType),
		Amount:    event.Amount.RawValue(),
	}
	if event.LotID != nil {
		accountEvent.LotID.String = string(*event.LotID)
		accountEvent.LotID.Valid = true
	}

	_, err := repo.client.NamedExec(query, &accountEvent)
	return errors.WithStack(err)
}

func (repo *userAccountEventRepository) FindAllByUserID(id app.UserID) ([]app.UserAccountEvent, error) {
	const query = `SELECT user_id, lot_id, event_type, amount FROM user_account_event WHERE user_id = $1 ORDER BY id`

	var events []*sqlxUserAccountEvent
	err := repo.client.Select(&events, query, string(id))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	res := make([]app.UserAccountEvent, 0, len(events))
	for _, event := range events {
		val := app.UserAccountEvent{
			UserID:    app.UserID(event.UserID),
			LotID:     nil,
			EventType: app.AccountEventType(event.EventType),
			Amount:    app.AmountFromRawValue(event.Amount),
		}
		if event.LotID.Valid {
			lotID := app.LotID(event.LotID.String)
			val.LotID = &lotID
		}
		res = append(res, val)
	}
	return res, nil
}

type sqlxUserAccountEvent struct {
	UserID    string         `db:"user_id"`
	LotID     sql.NullString `db:"lot_id"`
	EventType string         `db:"event_type"`
	Amount    uint64         `db:"amount"`
}
