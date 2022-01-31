package postgres

import (
	"arch-homework/pkg/billing/app"
	"arch-homework/pkg/common/infrastructure/postgres"

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

func (t *transactionalUnit) UserAccountEventRepository() app.UserAccountEventRepository {
	return NewUserAccountEventRepository(t.transaction)
}

func (t *transactionalUnit) ProcessedEventRepository() app.ProcessedEventRepository {
	return NewProcessedEventRepository(t.transaction)
}

func (t *transactionalUnit) ProcessedRequestRepository() app.ProcessedRequestRepository {
	return NewProcessedRequestRepository(t.transaction)
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
