package app

import "arch-homework/pkg/common/app/storedevent"

type RepositoryProvider interface {
	LotRepository() LotRepository
	BidRepository() BidRepository
	ProcessedRequestRepository() ProcessedRequestRepository
	ProcessedEventRepository() ProcessedEventRepository
	EventStore() storedevent.EventStore
}

type ReadRepositoryProvider interface {
	LotRepositoryRead() LotRepositoryRead
	ProcessedRequestRepositoryRead() ProcessedRequestRepositoryRead
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

type DBDependency interface {
	TransactionalUnitFactory
	ReadRepositoryProvider
}
