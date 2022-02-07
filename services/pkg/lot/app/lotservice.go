package app

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/storedevent"
	"arch-homework/pkg/common/app/uuid"

	"github.com/pkg/errors"

	"fmt"
	"time"
)

const minLotDurationAfterBid = time.Second * 60
const maxCancelAttempts = 10
const lotLockNameTpl = "lock_lot_%s"

var ErrBidOnOwnLot = errors.New("can't add bids for own lots")
var ErrPaymentFailed = errors.New("order payment failed")
var ErrInvalidEndTime = errors.New("invalid end time")
var ErrInvalidBuyItNowPrice = errors.New("invalid buy it now price")
var ErrLotClosed = errors.New("lot closed")
var ErrInvalidBidAmount = errors.New("invalid bid amount")
var ErrAlreadyProcessed = errors.New("request with this id already processed")

func NewLotService(
	readRepoProvider ReadRepositoryProvider,
	trUnitFactory TransactionalUnitFactory,
	eventSender storedevent.Sender,
	billingClient BillingClient,
) LotService {
	return &lotService{
		readRepoProvider: readRepoProvider,
		trUnitFactory:    trUnitFactory,
		eventSender:      eventSender,
		billingClient:    billingClient,
	}
}

type LotService interface {
	CreateLot(requestID RequestID, userID UserID, description string, startPrice float64, endTime time.Time, buyItNowPrice *float64) (LotID, error)
	CreateBid(requestID RequestID, userID UserID, lotID LotID, amount float64) error
	SetLotSent(lotID LotID) error
	SetLotReceived(lotID LotID) error
	ProcessCompletedLots() error
}

type lotService struct {
	readRepoProvider ReadRepositoryProvider
	trUnitFactory    TransactionalUnitFactory
	eventSender      storedevent.Sender
	billingClient    BillingClient
}

func (s *lotService) CreateLot(requestID RequestID, userID UserID, description string, startPrice float64, endTime time.Time, buyItNowPrice *float64) (LotID, error) {
	startPriceAmount, err := AmountFromFloat(startPrice)
	if err != nil {
		return "", err
	}
	var buyItNowAmount *Amount
	if buyItNowPrice != nil {
		amount, err := AmountFromFloat(*buyItNowPrice)
		if err != nil {
			return "", err
		}
		if amount.RawValue() < startPriceAmount.RawValue() {
			return "", errors.WithStack(ErrInvalidBuyItNowPrice)
		}
		buyItNowAmount = &amount
	}
	if !endTime.After(time.Now()) {
		return "", errors.WithStack(ErrInvalidEndTime)
	}

	lotID := LotID(uuid.GenerateNew())

	err = s.executeInTransactionWithLock("", func(provider RepositoryProvider) error {
		eventRepo := provider.ProcessedRequestRepository()
		alreadyProcessed, err2 := eventRepo.SetRequestProcessed(requestID)
		if err2 != nil {
			return err2
		}
		if alreadyProcessed {
			return errors.WithStack(ErrAlreadyProcessed)
		}

		lot := Lot{
			ID:            lotID,
			OwnerID:       userID,
			Description:   description,
			StartPrice:    startPriceAmount,
			BuyItNowPrice: buyItNowAmount,
			Status:        LotStatusActive,
			EndTime:       endTime,
			CreationTime:  time.Now(),
		}

		return provider.LotRepository().Store(&lot)
	})

	return lotID, err
}

func (s *lotService) CreateBid(requestID RequestID, userID UserID, lotID LotID, amount float64) error {
	bidAmount, err := AmountFromFloat(amount)
	if err != nil {
		return err
	}
	if err = s.checkRequestID(requestID); err != nil {
		return errors.WithStack(err)
	}

	paymentSucceeded, err := s.billingClient.ProcessOrderPayment(userID, lotID, bidAmount)
	if err != nil {
		return err
	}
	if !paymentSucceeded {
		return errors.WithStack(ErrPaymentFailed)
	}

	err = s.executeInTransactionWithLock(lotLockName(lotID), func(provider RepositoryProvider) error {
		eventRepo := provider.ProcessedRequestRepository()
		alreadyProcessed, err := eventRepo.SetRequestProcessed(requestID)
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return errors.WithStack(ErrAlreadyProcessed)
		}

		err = s.handleLotNewBid(provider, lotID, userID, bidAmount)
		if err != nil {
			return err
		}

		bidRepo := provider.BidRepository()
		lastBid, err := bidRepo.TryFindLastByLotID(lotID)
		if err != nil {
			return err
		}
		if lastBid != nil {
			if bidAmount.RawValue() <= lastBid.Amount.RawValue() {
				return errors.WithStack(ErrInvalidBidAmount)
			}
			event := NewBidOutbidEvent(lotID, lastBid.UserID, lastBid.Amount)
			err = provider.EventStore().Add(event)
			if err != nil {
				return err
			}
			s.eventSender.EventStored(event.UID)
		}

		bid := Bid{
			LotID:        lotID,
			UserID:       userID,
			Amount:       bidAmount,
			CreationTime: time.Now(),
		}
		return bidRepo.Store(&bid)
	})
	if err != nil {
		if paymentSucceeded {
			err2 := s.sendLotBidCancelledEvent(lotID, userID, bidAmount)
			if err2 != nil {
				err = errors.Wrap(err, err2.Error())
			}
		}
		return err
	}

	s.eventSender.SendStoredEvents()
	return nil
}

func (s *lotService) SetLotSent(lotID LotID) error {
	err := s.executeInTransactionWithLock(lotLockName(lotID), func(provider RepositoryProvider) error {
		lotRepo := provider.LotRepository()
		lot, err := lotRepo.FindByID(lotID)
		if err != nil {
			return err
		}

		bidRepo := provider.BidRepository()
		lastBid, err := bidRepo.TryFindLastByLotID(lotID)
		if err != nil {
			return err
		}
		if lastBid == nil {
			return errors.New("can't set status 'sent' for lot without bids")
		}

		event := NewLotSentEvent(lotID, lastBid.UserID, lot.OwnerID)
		err = provider.EventStore().Add(event)
		if err != nil {
			return err
		}
		s.eventSender.EventStored(event.UID)

		lot.Status = LotStatusSent
		return lotRepo.Store(lot)
	})
	if err != nil {
		return err
	}
	s.eventSender.SendStoredEvents()
	return nil
}

func (s *lotService) SetLotReceived(lotID LotID) error {
	err := s.executeInTransactionWithLock(lotLockName(lotID), func(provider RepositoryProvider) error {
		lotRepo := provider.LotRepository()
		lot, err := lotRepo.FindByID(lotID)
		if err != nil {
			return err
		}

		bidRepo := provider.BidRepository()
		lastBid, err := bidRepo.TryFindLastByLotID(lotID)
		if err != nil {
			return err
		}
		if lastBid == nil {
			return errors.New("can't set status 'received' for lot without bids")
		}

		event := NewLotReceivedEvent(lotID, lastBid.UserID, lot.OwnerID, lastBid.Amount)
		err = provider.EventStore().Add(event)
		if err != nil {
			return err
		}
		s.eventSender.EventStored(event.UID)

		lot.Status = LotStatusReceived
		return lotRepo.Store(lot)
	})
	if err != nil {
		return err
	}
	s.eventSender.SendStoredEvents()
	return nil
}

func (s *lotService) ProcessCompletedLots() error {
	lots, err := s.readRepoProvider.LotRepositoryRead().FindActiveCompletedLots()
	if err != nil || len(lots) == 0 {
		return err
	}

	for _, completedLot := range lots {
		lotID := completedLot.ID
		endTime := completedLot.EndTime

		err = s.executeInTransactionWithLock(lotLockName(lotID), func(provider RepositoryProvider) error {
			lotRepo := provider.LotRepository()
			lot, err := lotRepo.FindByID(lotID)
			if err != nil {
				return err
			}
			if lot.EndTime.After(endTime) {
				// lot time extended
				return nil
			}

			bidRepo := provider.BidRepository()
			lastBid, err := bidRepo.TryFindLastByLotID(lotID)
			if err != nil {
				return err
			}
			var event integrationevent.EventData
			if lastBid != nil {
				lot.Status = LotStatusFinished
				event = NewLotWonEvent(lotID, lastBid.UserID, lot.OwnerID)
			} else {
				lot.Status = LotStatusClosed
				event = NewLotClosedEvent(lotID, lot.OwnerID)
			}
			err = provider.EventStore().Add(event)
			if err != nil {
				return err
			}
			s.eventSender.EventStored(event.UID)

			return lotRepo.Store(lot)
		})
		if err != nil {
			break
		}
	}
	s.eventSender.SendStoredEvents()
	return err
}

func (s *lotService) handleLotNewBid(provider RepositoryProvider, lotID LotID, userID UserID, bidAmount Amount) error {
	lotRepo := provider.LotRepository()
	lot, err := lotRepo.FindByID(lotID)
	if err != nil {
		return err
	}
	if lot.OwnerID == userID {
		return errors.WithStack(ErrBidOnOwnLot)
	}
	if bidAmount.RawValue() < lot.StartPrice.RawValue() {
		return errors.WithStack(ErrInvalidBidAmount)
	}
	curTime := time.Now()
	if lot.EndTime.Before(curTime) || lot.Status != LotStatusActive {
		return errors.WithStack(ErrLotClosed)
	}
	lotChanged := false
	minLotEndTime := curTime.Add(minLotDurationAfterBid)
	if lot.EndTime.Before(minLotEndTime) {
		lot.EndTime = minLotEndTime
		lotChanged = true
	}

	if lot.BuyItNowPrice != nil && bidAmount.RawValue() >= (*lot.BuyItNowPrice).RawValue() {
		lot.Status = LotStatusFinished
		lotChanged = true

		event := NewLotWonEvent(lotID, userID, lot.OwnerID)
		err = provider.EventStore().Add(event)
		if err != nil {
			return err
		}
		s.eventSender.EventStored(event.UID)
	}
	if lotChanged {
		err = lotRepo.Store(lot)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *lotService) sendLotBidCancelledEvent(lotID LotID, userID UserID, bidAmount Amount) error {
	var err error
	for i := 0; i < maxCancelAttempts; i++ {
		err = s.executeInTransactionWithLock(lotLockName(lotID), func(provider RepositoryProvider) error {
			event := NewBidCancelledEvent(lotID, userID, bidAmount)

			err := provider.EventStore().Add(event)
			if err != nil {
				return err
			}
			s.eventSender.EventStored(event.UID)

			return nil
		})
		if err == nil {
			s.eventSender.SendStoredEvents()
			return nil
		}
	}
	return err
}

func (s *lotService) checkRequestID(requestID RequestID) error {
	processed, err := s.readRepoProvider.ProcessedRequestRepositoryRead().IsRequestProcessed(requestID)
	if err != nil {
		return err
	}
	if processed {
		return ErrAlreadyProcessed
	}
	return nil
}

func (s *lotService) executeInTransactionWithLock(lockName string, f func(RepositoryProvider) error) (err error) {
	var trUnit TransactionalUnit
	trUnit, err = s.trUnitFactory.NewTransactionalUnit()
	if err != nil {
		return err
	}
	defer func() {
		err = trUnit.Complete(err)
	}()
	if lockName != "" {
		err = trUnit.AddLock(lockName)
		if err != nil {
			return err
		}
	}
	err = f(trUnit)
	return err
}

func lotLockName(lotID LotID) string {
	return fmt.Sprintf(lotLockNameTpl, string(lotID))
}
