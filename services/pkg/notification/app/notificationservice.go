package app

import (
	"fmt"

	"github.com/pkg/errors"
)

func NewNotificationService(repo NotificationRepository) NotificationService {
	return &notificationService{
		repo: repo,
	}
}

type NotificationService interface {
	AddNotification(notificationType NotificationType, lotID LotID, userID UserID) error
}

type notificationService struct {
	repo NotificationRepository
}

func (n *notificationService) AddNotification(notificationType NotificationType, lotID LotID, userID UserID) error {
	msg, err := messageForLot(notificationType, lotID)
	if err != nil {
		return errors.WithStack(err)
	}
	notification := Notification{
		Type:    notificationType,
		UserID:  userID,
		LotID:   &lotID,
		Message: msg,
	}
	return n.repo.Store(&notification)
}

func messageForLot(notificationType NotificationType, lotID LotID) (string, error) {
	switch notificationType {
	case TypeLotFinished:
		return fmt.Sprintf("Your lot %s successfully finished", string(lotID)), nil
	case TypeLotClosed:
		return fmt.Sprintf("Your lot %s closed without bids", string(lotID)), nil
	case TypeLotWon:
		return fmt.Sprintf("You won the lot %s", string(lotID)), nil
	case TypeLotSent:
		return fmt.Sprintf("Lot %s has been sent", string(lotID)), nil
	case TypeLotReceived:
		return fmt.Sprintf("Lot %s has been received", string(lotID)), nil
	case TypeBidOutbid:
		return fmt.Sprintf("Your bid in the lot %s has been outbid", string(lotID)), nil
	default:
		return "", errors.New("unknown notification type")
	}
}
