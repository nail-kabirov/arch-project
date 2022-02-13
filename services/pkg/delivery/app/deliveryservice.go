package app

import (
	"arch-homework/pkg/common/app/storedevent"

	"github.com/pkg/errors"
)

var ErrAlreadyProcessed = errors.New("request with this id already processed")
var ErrInvalidLotStatus = errors.New("lot status is invalid for this operation")
var ErrStatusChangeForbidden = errors.New("delivery status change forbidden for this user")

func NewDeliveryService(dbDependency DBDependency, eventSender storedevent.Sender, lotSvcClient LotServiceClient, userSvcClient UserServiceClient) *DeliveryService {
	return &DeliveryService{
		readRepo:      dbDependency.DeliveryInfoRepositoryRead(),
		trUnitFactory: dbDependency,
		eventSender:   eventSender,
		lotSvcClient:  lotSvcClient,
		userSvcClient: userSvcClient,
	}
}

type DeliveryService struct {
	readRepo      DeliveryInfoRepositoryRead
	trUnitFactory TransactionalUnitFactory
	eventSender   storedevent.Sender
	lotSvcClient  LotServiceClient
	userSvcClient UserServiceClient
}

func (s *DeliveryService) LotDeliveryInfo(lotID LotID) (*DeliveryInfo, error) {
	info, err := s.readRepo.FindByLotID(lotID)
	if err == nil {
		return info, nil
	}
	if errors.Cause(err) != ErrLotNotFound {
		return nil, err
	}
	return s.deliveryInfoFromServices(lotID)
}

func (s *DeliveryService) SetLotSent(requestID RequestID, userID UserID, lotID LotID, trackingID TrackingID) error {
	deliveryInfo, err := s.deliveryInfoFromServices(lotID)
	if err != nil {
		return err
	}

	if deliveryInfo.SenderID != userID {
		return errors.WithStack(ErrStatusChangeForbidden)
	}

	err = s.executeInTransaction(func(provider RepositoryProvider) error {
		eventRepo := provider.ProcessedRequestRepository()
		alreadyProcessed, err := eventRepo.SetRequestProcessed(requestID)
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return ErrAlreadyProcessed
		}

		deliveryInfoRepo := provider.DeliveryInfoRepository()
		deliveryInfo.LotStatus = LotStatusSent
		deliveryInfo.TrackingID = &trackingID
		err = deliveryInfoRepo.Store(deliveryInfo)
		if err != nil {
			return err
		}

		event := NewLotSentEvent(lotID)
		err = provider.EventStore().Add(event)
		if err != nil {
			return err
		}
		s.eventSender.EventStored(event.UID)
		return nil
	})
	if err != nil {
		return err
	}
	s.eventSender.SendStoredEvents()
	return nil
}

func (s *DeliveryService) SetLotReceived(requestID RequestID, userID UserID, lotID LotID) error {
	err := s.executeInTransaction(func(provider RepositoryProvider) error {
		eventRepo := provider.ProcessedRequestRepository()
		alreadyProcessed, err := eventRepo.SetRequestProcessed(requestID)
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return ErrAlreadyProcessed
		}

		deliveryInfoRepo := provider.DeliveryInfoRepository()
		info, err := deliveryInfoRepo.FindByLotID(lotID)
		if err != nil {
			return err
		}
		if info.LotStatus != LotStatusSent {
			return errors.WithStack(ErrInvalidLotStatus)
		}
		if info.ReceiverID != userID {
			return errors.WithStack(ErrStatusChangeForbidden)
		}

		event := NewLotReceivedEvent(lotID)
		err = provider.EventStore().Add(event)
		if err != nil {
			return err
		}
		s.eventSender.EventStored(event.UID)

		info.LotStatus = LotStatusReceived
		return deliveryInfoRepo.Store(info)
	})
	if err != nil {
		return err
	}
	s.eventSender.SendStoredEvents()
	return nil
}

func (s *DeliveryService) deliveryInfoFromServices(lotID LotID) (*DeliveryInfo, error) {
	lotInfo, err := s.lotSvcClient.FindFinishedLotInfo(lotID)
	if err != nil {
		return nil, err
	}
	ownerInfo, err := s.userSvcClient.GetUserInfo(lotInfo.OwnerID)
	if err != nil {
		return nil, err
	}
	receiverInfo, err := s.userSvcClient.GetUserInfo(lotInfo.ReceiverID)
	if err != nil {
		return nil, err
	}
	deliveryInfo := DeliveryInfo{
		LotID:             lotInfo.ID,
		LotStatus:         LotStatusFinished,
		TrackingID:        nil,
		ReceiverID:        lotInfo.ReceiverID,
		ReceiverLogin:     receiverInfo.Login,
		ReceiverFirstName: receiverInfo.FirstName,
		ReceiverLastName:  receiverInfo.LastName,
		ReceiverAddress:   Address(receiverInfo.Address),
		SenderID:          lotInfo.OwnerID,
		SenderLogin:       ownerInfo.Login,
		SenderFirstName:   ownerInfo.FirstName,
		SenderLastName:    ownerInfo.LastName,
	}
	return &deliveryInfo, nil
}

func (s *DeliveryService) executeInTransaction(f func(RepositoryProvider) error) (err error) {
	var trUnit TransactionalUnit
	trUnit, err = s.trUnitFactory.NewTransactionalUnit()
	if err != nil {
		return err
	}
	defer func() {
		err = trUnit.Complete(err)
	}()
	err = f(trUnit)
	return err
}
