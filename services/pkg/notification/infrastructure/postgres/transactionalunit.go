package postgres

import (
	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/notification/app"

	"github.com/pkg/errors"
)

func NewTransactionalUnitFactory(client postgres.TransactionalClient) app.TransactionalUnitFactory {
	return &transactionalUnitFactory{client: client}
}

type transactionalUnitFactory struct {
	client postgres.TransactionalClient
}

func (d *transactionalUnitFactory) NewTransactionalUnit() (app.TransactionalUnit, error) {
	transaction, err := d.client.BeginTransaction()
	if err != nil {
		return nil, err
	}
	return &transactionalUnit{transaction: transaction}, nil
}

type transactionalUnit struct {
	transaction postgres.Transaction
}

func (t *transactionalUnit) NotificationRepository() app.NotificationRepository {
	return NewNotificationRepository(t.transaction)
}

func (t *transactionalUnit) ProcessedEventRepo() app.ProcessedEventRepository {
	return NewProcessedEventRepository(t.transaction)
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
