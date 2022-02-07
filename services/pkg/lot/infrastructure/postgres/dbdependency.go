package postgres

import (
	"arch-homework/pkg/common/app/storedevent"
	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/lot/app"

	"github.com/pkg/errors"
)

func NewDBDependency(client postgres.TransactionalClient) app.DBDependency {
	return &dbDependency{client: client}
}

type dbDependency struct {
	client postgres.TransactionalClient
}

func (d *dbDependency) ProcessedRequestRepositoryRead() app.ProcessedRequestRepositoryRead {
	return NewProcessedRequestRepository(d.client)
}

func (d *dbDependency) LotRepositoryRead() app.LotRepositoryRead {
	return NewLotRepository(d.client)
}

func (d *dbDependency) NewTransactionalUnit() (app.TransactionalUnit, error) {
	transaction, err := d.client.BeginTransaction()
	if err != nil {
		return nil, err
	}
	return &transactionalUnit{transaction: transaction, nestedLevel: 1}, nil
}

type transactionalUnit struct {
	transaction postgres.Transaction
	nestedLevel uint
	completeErr error
}

func (t *transactionalUnit) NewTransactionalUnit() (app.TransactionalUnit, error) {
	t.nestedLevel++
	return t, nil
}

func (t *transactionalUnit) BidRepository() app.BidRepository {
	return NewBidRepository(t.transaction)
}

func (t *transactionalUnit) EventStore() storedevent.EventStore {
	return NewEventStore(t.transaction)
}

func (t *transactionalUnit) LotRepository() app.LotRepository {
	return NewLotRepository(t.transaction)
}

func (t *transactionalUnit) ProcessedRequestRepository() app.ProcessedRequestRepository {
	return NewProcessedRequestRepository(t.transaction)
}

func (t *transactionalUnit) ProcessedEventRepository() app.ProcessedEventRepository {
	return NewProcessedEventRepository(t.transaction)
}

func (t *transactionalUnit) Complete(err error) error {
	t.nestedLevel--

	if t.completeErr != nil {
		return t.completeErr
	}

	if err != nil {
		rollbackErr := t.transaction.Rollback()
		if rollbackErr != nil {
			err = errors.Wrap(err, rollbackErr.Error())
		}
		t.completeErr = err
		return err
	}
	if t.nestedLevel > 0 {
		return nil
	}
	return errors.WithStack(t.transaction.Commit())
}

func (t *transactionalUnit) AddLock(lockName string) error {
	const lockQuery = "SELECT pg_advisory_xact_lock(hashtext($1))"
	_, err := t.transaction.Exec(lockQuery, lockName)
	return errors.WithStack(err)
}
