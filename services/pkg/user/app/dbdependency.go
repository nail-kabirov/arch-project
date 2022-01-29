package app

import "arch-homework/pkg/common/app/storedevent"

type RepositoryProvider interface {
	UserProfileRepository() UserProfileRepository
	ProcessedRequestRepository() ProcessedRequestRepository
	EventStore() storedevent.EventStore
}

type ReadRepositoryProvider interface {
	UserProfileRepositoryRead() UserProfileRepositoryRead
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
