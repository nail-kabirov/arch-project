package postgres

import (
	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/delivery/app"

	"database/sql"

	"github.com/pkg/errors"
)

func NewDeliveryInfoRepository(client postgres.Client) app.DeliveryInfoRepository {
	return &deliveryInfoRepository{client: client}
}

type deliveryInfoRepository struct {
	client postgres.Client
}

func (repo *deliveryInfoRepository) FindByLotID(id app.LotID) (*app.DeliveryInfo, error) {
	const query = `
			SELECT lot_id, status, tracking_id, receiver_id, receiver_login, receiver_first_name, receiver_last_name,
				receiver_address, sender_id, sender_login, sender_first_name, sender_last_name
			FROM delivery WHERE lot_id = $1
		`

	var info sqlxDeliveryInfo
	err := repo.client.Get(&info, query, string(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrLotNotFound
		}
		return nil, errors.WithStack(err)
	}
	res := sqlxDeliveryInfoToDeliveryInfo(info)
	return &res, nil
}

func (repo *deliveryInfoRepository) Store(info *app.DeliveryInfo) error {
	const query = `
			INSERT INTO delivery (lot_id, status, tracking_id, receiver_id, receiver_login, receiver_first_name, receiver_last_name, receiver_address, sender_id, sender_login, sender_first_name, sender_last_name)
			VALUES (:lot_id, :status, :tracking_id, :receiver_id, :receiver_login, :receiver_first_name, :receiver_last_name, :receiver_address, :sender_id, :sender_login, :sender_first_name, :sender_last_name)
			ON CONFLICT (lot_id) DO UPDATE SET
				status = excluded.status,
				tracking_id = excluded.tracking_id,
				receiver_login = excluded.receiver_login,
				receiver_first_name = excluded.receiver_first_name,
				receiver_last_name = excluded.receiver_last_name,
				receiver_address = excluded.receiver_address,
				sender_login = excluded.sender_login,
				sender_first_name = excluded.sender_first_name,
				sender_last_name = excluded.sender_last_name;
		`

	if info.TrackingID == nil {
		return errors.WithStack(errors.New("can't save delivery info with empty tracking id"))
	}

	infox := sqlxDeliveryInfo{
		LotID:             string(info.LotID),
		Status:            string(info.LotStatus),
		TrackingID:        string(*info.TrackingID),
		ReceiverID:        string(info.ReceiverID),
		ReceiverLogin:     info.ReceiverLogin,
		ReceiverFirstName: info.ReceiverFirstName,
		ReceiverLastName:  info.ReceiverLastName,
		ReceiverAddress:   string(info.ReceiverAddress),
		SenderID:          string(info.SenderID),
		SenderLogin:       info.SenderLogin,
		SenderFirstName:   info.SenderFirstName,
		SenderLastName:    info.SenderLastName,
	}

	_, err := repo.client.NamedExec(query, &infox)
	return errors.WithStack(err)
}

func sqlxDeliveryInfoToDeliveryInfo(info sqlxDeliveryInfo) app.DeliveryInfo {
	trackingID := app.TrackingID(info.TrackingID)
	return app.DeliveryInfo{
		LotID:             app.LotID(info.LotID),
		LotStatus:         app.LotStatus(info.Status),
		TrackingID:        &trackingID,
		ReceiverID:        app.UserID(info.ReceiverID),
		ReceiverLogin:     info.ReceiverLogin,
		ReceiverFirstName: info.ReceiverFirstName,
		ReceiverLastName:  info.ReceiverLastName,
		ReceiverAddress:   app.Address(info.ReceiverAddress),
		SenderID:          app.UserID(info.SenderID),
		SenderLogin:       info.SenderLogin,
		SenderFirstName:   info.SenderFirstName,
		SenderLastName:    info.SenderLastName,
	}
}

type sqlxDeliveryInfo struct {
	LotID             string `db:"lot_id"`
	Status            string `db:"status"`
	TrackingID        string `db:"tracking_id"`
	ReceiverID        string `db:"receiver_id"`
	ReceiverLogin     string `db:"receiver_login"`
	ReceiverFirstName string `db:"receiver_first_name"`
	ReceiverLastName  string `db:"receiver_last_name"`
	ReceiverAddress   string `db:"receiver_address"`
	SenderID          string `db:"sender_id"`
	SenderLogin       string `db:"sender_login"`
	SenderFirstName   string `db:"sender_first_name"`
	SenderLastName    string `db:"sender_last_name"`
}
