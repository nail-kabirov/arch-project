package postgres

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/lot/app"
)

func NewLotRepository(client postgres.Client) app.LotRepository {
	return &lotRepository{client: client}
}

type lotRepository struct {
	client postgres.Client
}

func (repo *lotRepository) FindByID(id app.LotID) (*app.Lot, error) {
	const query = `SELECT id, owner_id, description, status, start_price, buy_it_now_price, end_time, created_at FROM lot WHERE id = $1`

	var lot sqlxLot
	err := repo.client.Get(&lot, query, string(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrLotNotFound
		}
		return nil, errors.WithStack(err)
	}
	res := sqlxLotToLot(&lot)
	return &res, nil
}

func (repo *lotRepository) FindActiveCompletedLots() ([]app.Lot, error) {
	const query = `
			SELECT id, owner_id, description, status, start_price, buy_it_now_price, end_time, created_at FROM lot
			WHERE status = $1 AND end_time < $2
		`

	var lots []*sqlxLot
	err := repo.client.Select(&lots, query, string(app.LotStatusActive), time.Now())
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make([]app.Lot, 0, len(lots))
	for _, lot := range lots {
		res = append(res, sqlxLotToLot(lot))
	}
	return res, nil
}

func (repo *lotRepository) Store(lot *app.Lot) error {
	const query = `
			INSERT INTO lot (id, owner_id, description, status, start_price, buy_it_now_price, end_time, created_at)
			VALUES (:id, :owner_id, :description, :status, :start_price, :buy_it_now_price, :end_time, :created_at)
			ON CONFLICT (id) DO UPDATE SET
				description = excluded.description,
				status = excluded.status,
				end_time = excluded.end_time;
		`

	lotx := sqlxLot{
		ID:           string(lot.ID),
		OwnerID:      string(lot.OwnerID),
		Description:  lot.Description,
		Status:       string(lot.Status),
		StartPrice:   lot.StartPrice.RawValue(),
		EndTime:      lot.EndTime,
		CreationTime: lot.CreationTime,
	}
	if lot.BuyItNowPrice != nil {
		lotx.BuyItNowPrice.Int64 = int64((*lot.BuyItNowPrice).RawValue())
		lotx.BuyItNowPrice.Valid = true
	}

	_, err := repo.client.NamedExec(query, &lotx)
	return errors.WithStack(err)
}

func sqlxLotToLot(lot *sqlxLot) app.Lot {
	var buyItNowPrice *app.Amount
	if lot.BuyItNowPrice.Valid {
		price := app.AmountFromRawValue(uint64(lot.BuyItNowPrice.Int64))
		buyItNowPrice = &price
	}

	return app.Lot{
		ID:            app.LotID(lot.ID),
		OwnerID:       app.UserID(lot.OwnerID),
		Description:   lot.Description,
		StartPrice:    app.AmountFromRawValue(lot.StartPrice),
		BuyItNowPrice: buyItNowPrice,
		Status:        app.LotStatus(lot.Status),
		EndTime:       lot.EndTime,
		CreationTime:  lot.CreationTime,
	}
}

type sqlxLot struct {
	ID            string        `db:"id"`
	OwnerID       string        `db:"owner_id"`
	Description   string        `db:"description"`
	Status        string        `db:"status"`
	StartPrice    uint64        `db:"start_price"`
	BuyItNowPrice sql.NullInt64 `db:"buy_it_now_price"`
	EndTime       time.Time     `db:"end_time"`
	CreationTime  time.Time     `db:"created_at"`
}
