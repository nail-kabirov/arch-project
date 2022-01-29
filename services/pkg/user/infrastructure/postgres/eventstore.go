package postgres

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/storedevent"
	"arch-homework/pkg/common/infrastructure/postgres"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"time"
)

func NewEventStore(client postgres.Client) storedevent.EventStore {
	return &eventStore{client: client}
}

type eventStore struct {
	client postgres.Client
}

func (store *eventStore) Add(event integrationevent.EventData) error {
	const query = `
			INSERT INTO stored_event (uid, type, body, confirmed)
			VALUES (:uid, :type, :body, :confirmed)
		`

	eventX := sqlxStoredEvent{
		UID:       string(event.UID),
		Type:      event.Type,
		Body:      event.Body,
		Confirmed: false,
	}

	_, err := store.client.NamedExec(query, &eventX)
	return errors.WithStack(err)
}

func (store *eventStore) ConfirmDelivery(id storedevent.EventID) error {
	const query = `UPDATE stored_event SET confirmed = TRUE WHERE id = $1`

	_, err := store.client.Exec(query, id)
	return errors.WithStack(err)
}

func (store *eventStore) FindByUIDs(uids []integrationevent.EventUID) ([]storedevent.Event, error) {
	const sqlQuery = `SELECT id, uid, type, body, confirmed FROM stored_event WHERE uid IN (?)`

	strUids := make([]string, 0, len(uids))
	for _, uid := range uids {
		strUids = append(strUids, string(uid))
	}

	query, params, err := sqlx.In(sqlQuery, strUids)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	query = sqlx.Rebind(sqlx.DOLLAR, query)

	var events []*sqlxStoredEvent
	err = store.client.Select(&events, query, params...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]storedevent.Event, 0, len(events))
	for _, event := range events {
		res = append(res, sqlxStoredEventToEvent(event))
	}
	return res, nil
}

func (store *eventStore) FindAllUnconfirmedBefore(time time.Time) ([]storedevent.Event, error) {
	const sqlQuery = `SELECT id, uid, type, body, confirmed FROM stored_event WHERE confirmed = FALSE AND created_at < $1`

	var events []*sqlxStoredEvent
	err := store.client.Select(&events, sqlQuery, time)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]storedevent.Event, 0, len(events))
	for _, event := range events {
		res = append(res, sqlxStoredEventToEvent(event))
	}
	return res, nil
}

func sqlxStoredEventToEvent(event *sqlxStoredEvent) storedevent.Event {
	return storedevent.Event{
		EventData: integrationevent.EventData{
			UID:  integrationevent.EventUID(event.UID),
			Type: event.Type,
			Body: event.Body,
		},
		ID:        storedevent.EventID(event.ID),
		Confirmed: event.Confirmed,
	}
}

type sqlxStoredEvent struct {
	ID        uint64 `db:"id"`
	UID       string `db:"uid"`
	Type      string `db:"type"`
	Body      string `db:"body"`
	Confirmed bool   `db:"confirmed"`
}
