package app

type RepositoryProvider interface {
	UserAccountEventRepository() UserAccountEventRepository
	ProcessedEventRepository() ProcessedEventRepository
	ProcessedRequestRepository() ProcessedRequestRepository
}

type TransactionalUnit interface {
	RepositoryProvider
	TransactionalUnitFactory
	Complete(err error) error
	AddLock(lockName string) error
}

type TransactionalUnitFactory interface {
	NewTransactionalUnit() (TransactionalUnit, error)
}
