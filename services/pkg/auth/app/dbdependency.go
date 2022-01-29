package app

type RepositoryProvider interface {
	UserRepository() UserRepository
}

type ReadRepositoryProvider interface {
	UserRepositoryRead() UserRepositoryRead
}

type TransactionalUnit interface {
	RepositoryProvider
	Complete(err error) error
}

type TransactionalUnitFactory interface {
	NewTransactionalUnit() (TransactionalUnit, error)
}

type DBDependency interface {
	TransactionalUnitFactory
	ReadRepositoryProvider
}
