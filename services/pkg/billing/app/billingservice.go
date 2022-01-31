package app

import (
	"fmt"
	"sort"

	"github.com/pkg/errors"
)

const userAccountEventLockNameTpl = "lock_acc_event_%s"

var ErrAlreadyProcessed = errors.New("request with this id already processed")

func NewBillingService(trUnitFactory TransactionalUnitFactory) BillingService {
	return &billingService{trUnitFactory: trUnitFactory}
}

type BillingService interface {
	CreateAccount(userID UserID) error
	CancelLotPayment(userID UserID, lotID LotID, amount Amount) error
	FinalizeLotPayment(lotOwnerID, winnerID UserID, lotID LotID, amount Amount) error

	TopUpAccount(requestID RequestID, userID UserID, amount Amount) error
	ProcessLotPayment(requestID RequestID, userID UserID, lotID LotID, amount Amount) error
}

type billingService struct {
	trUnitFactory TransactionalUnitFactory
}

func (s *billingService) CancelLotPayment(userID UserID, lotID LotID, amount Amount) error {
	return s.executeInTransactionWithLock(
		[]string{userAccountEventLockName(userID)},
		func(repoProvider RepositoryProvider) error {
			return s.changeAccountState(
				repoProvider.UserAccountEventRepository(),
				userID,
				func(state UserAccountState) error {
					return state.AddUnblockPaymentEvent(lotID, amount)
				})
		})
}

func (s *billingService) FinalizeLotPayment(lotOwnerID, winnerID UserID, lotID LotID, amount Amount) error {
	return s.executeInTransactionWithLock(
		[]string{userAccountEventLockName(lotOwnerID), userAccountEventLockName(winnerID)},
		func(repoProvider RepositoryProvider) error {
			err := s.changeAccountState(
				repoProvider.UserAccountEventRepository(),
				winnerID,
				func(state UserAccountState) error {
					return state.AddFinishPaymentEvent(lotID, amount)
				})
			if err != nil {
				return err
			}

			return s.changeAccountState(
				repoProvider.UserAccountEventRepository(),
				lotOwnerID,
				func(state UserAccountState) error {
					return state.AddReceivePaymentEvent(lotID, amount)
				})
		})
}

func (s *billingService) CreateAccount(userID UserID) error {
	return s.executeInTransactionWithLock(
		[]string{userAccountEventLockName(userID)},
		func(repoProvider RepositoryProvider) error {
			return s.changeAccountState(
				repoProvider.UserAccountEventRepository(),
				userID,
				func(state UserAccountState) error {
					return state.AddCreateAccountEvent()
				})
		})
}

func (s *billingService) TopUpAccount(requestID RequestID, userID UserID, amount Amount) error {
	return s.executeInTransactionWithLock(
		[]string{userAccountEventLockName(userID)},
		func(provider RepositoryProvider) error {
			err := s.checkRequestProcessed(provider.ProcessedRequestRepository(), requestID)
			if err != nil {
				return err
			}

			return s.changeAccountState(
				provider.UserAccountEventRepository(),
				userID,
				func(state UserAccountState) error {
					return state.AddTopUpAccountEvent(amount)
				})
		})
}

func (s *billingService) ProcessLotPayment(requestID RequestID, userID UserID, lotID LotID, amount Amount) error {
	return s.executeInTransactionWithLock(
		[]string{userAccountEventLockName(userID)},
		func(provider RepositoryProvider) error {
			err := s.checkRequestProcessed(provider.ProcessedRequestRepository(), requestID)
			if err != nil {
				return err
			}

			return s.changeAccountState(
				provider.UserAccountEventRepository(),
				userID,
				func(state UserAccountState) error {
					return state.AddBlockPaymentEvent(lotID, amount)
				})
		})
}

func (s *billingService) checkRequestProcessed(requestRepo ProcessedRequestRepository, requestID RequestID) error {
	alreadyProcessed, err := requestRepo.SetRequestProcessed(requestID)
	if err != nil {
		return err
	}
	if alreadyProcessed {
		return errors.WithStack(ErrAlreadyProcessed)
	}
	return nil
}

func (s *billingService) changeAccountState(accountEventRepo UserAccountEventRepository, userID UserID, f func(UserAccountState) error) error {
	accountEvents, err := accountEventRepo.FindAllByUserID(userID)
	if err != nil {
		return err
	}

	state := NewEmptyUserAccountState(userID)
	err = state.LoadEvents(accountEvents)
	if err != nil {
		return err
	}

	err = f(state)
	if err != nil {
		return err
	}

	addedEvents := state.AddedEvents()
	for _, event := range addedEvents {
		event := event
		err = accountEventRepo.Store(&event)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *billingService) executeInTransactionWithLock(lockNames []string, f func(RepositoryProvider) error) (err error) {
	var trUnit TransactionalUnit
	trUnit, err = s.trUnitFactory.NewTransactionalUnit()
	if err != nil {
		return err
	}
	defer func() {
		err = trUnit.Complete(err)
	}()
	if len(lockNames) > 0 {
		// sort for prevent possible deadlocks
		sort.Strings(lockNames)

		for _, lockName := range lockNames {
			err = trUnit.AddLock(lockName)
			if err != nil {
				return err
			}
		}
	}
	err = f(trUnit)
	return err
}

func userAccountEventLockName(userID UserID) string {
	return fmt.Sprintf(userAccountEventLockNameTpl, string(userID))
}
