package postgres

import (
	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/notification/app"
	"database/sql"
	"time"

	"github.com/pkg/errors"
)

func NewNotificationRepository(client postgres.Client) app.NotificationRepository {
	return &notificationRepository{client: client}
}

type notificationRepository struct {
	client postgres.Client
}

func (repo *notificationRepository) Store(notification *app.Notification) error {
	const query = `
			INSERT INTO notification (type, user_id, lot_id, message)
			VALUES (:type, :user_id, :lot_id, :message)	
		`

	notificationx := sqlxNotification{
		Type:    string(notification.Type),
		UserID:  string(notification.UserID),
		Message: notification.Message,
	}
	if notification.LotID != nil {
		notificationx.LotID.String = string(*notification.LotID)
		notificationx.LotID.Valid = true
	}

	_, err := repo.client.NamedExec(query, &notificationx)
	return errors.WithStack(err)
}

func (repo *notificationRepository) FindAllByUserID(userID app.UserID) ([]app.Notification, error) {
	const query = `SELECT type, user_id, lot_id, message, created_at FROM notification where user_id = $1 ORDER BY id`

	var notifications []*sqlxNotification
	err := repo.client.Select(&notifications, query, string(userID))
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]app.Notification, 0, len(notifications))
	for _, notification := range notifications {
		res = append(res, sqlxNotificationToNotification(notification))
	}
	return res, nil
}

func sqlxNotificationToNotification(notification *sqlxNotification) app.Notification {
	var lotID *app.LotID
	if notification.LotID.Valid {
		id := app.LotID(notification.LotID.String)
		lotID = &id
	}

	return app.Notification{
		Type:         app.NotificationType(notification.Type),
		UserID:       app.UserID(notification.UserID),
		LotID:        lotID,
		Message:      notification.Message,
		CreationDate: notification.CreationDate,
	}
}

type sqlxNotification struct {
	Type         string         `db:"type"`
	UserID       string         `db:"user_id"`
	LotID        sql.NullString `db:"lot_id"`
	Message      string         `db:"message"`
	CreationDate time.Time      `db:"created_at"`
}
