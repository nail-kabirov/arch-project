package postgres

import (
	"arch-homework/pkg/auth/app"
	"arch-homework/pkg/common/infrastructure/postgres"

	"github.com/pkg/errors"
)

func NewDBDependency(client postgres.TransactionalClient) app.DBDependency {
	return &dbDependency{client: client}
}

type dbDependency struct {
	client postgres.TransactionalClient
}

func (d *dbDependency) NewTransactionalUnit() (app.TransactionalUnit, error) {
	transaction, err := d.client.BeginTransaction()
	if err != nil {
		return nil, err
	}
	return &transactionalUnit{transaction: transaction}, nil
}

func (d *dbDependency) UserRepositoryRead() app.UserRepositoryRead {
	return NewUserRepository(d.client)
}

type transactionalUnit struct {
	transaction postgres.Transaction
}

func (t *transactionalUnit) UserRepository() app.UserRepository {
	return NewUserRepository(t.transaction)
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
