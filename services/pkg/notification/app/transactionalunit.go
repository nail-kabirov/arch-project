package app

type RepositoryProvider interface {
	NotificationRepository() NotificationRepository
	ProcessedEventRepo() ProcessedEventRepository
}

type TransactionalUnit interface {
	RepositoryProvider
	Complete(err error) error
}

type TransactionalUnitFactory interface {
	NewTransactionalUnit() (TransactionalUnit, error)
}
