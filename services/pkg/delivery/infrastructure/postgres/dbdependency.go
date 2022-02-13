package postgres

import (
	"arch-homework/pkg/common/app/storedevent"
	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/delivery/app"

	"github.com/pkg/errors"
)

func NewDBDependency(client postgres.TransactionalClient) app.DBDependency {
	return &dbDependency{client: client}
}

type dbDependency struct {
	client postgres.TransactionalClient
}

func (d *dbDependency) DeliveryInfoRepositoryRead() app.DeliveryInfoRepositoryRead {
	return NewDeliveryInfoRepository(d.client)
}

func (d *dbDependency) NewTransactionalUnit() (app.TransactionalUnit, error) {
	transaction, err := d.client.BeginTransaction()
	if err != nil {
		return nil, err
	}
	return &transactionalUnit{transaction: transaction}, nil
}

type transactionalUnit struct {
	transaction postgres.Transaction
}

func (t *transactionalUnit) DeliveryInfoRepository() app.DeliveryInfoRepository {
	return NewDeliveryInfoRepository(t.transaction)
}

func (t *transactionalUnit) EventStore() storedevent.EventStore {
	return NewEventStore(t.transaction)
}

func (t *transactionalUnit) ProcessedRequestRepository() app.ProcessedRequestRepository {
	return NewProcessedRequestRepository(t.transaction)
}

func (t *transactionalUnit) Complete(err error) error {
	if err != nil {
		rollbackErr := t.transaction.Rollback()
		if rollbackErr != nil {
			return errors.Wrap(err, rollbackErr.Error())
		}
		return err
	}

	return errors.WithStack(t.transaction.Commit())
}
