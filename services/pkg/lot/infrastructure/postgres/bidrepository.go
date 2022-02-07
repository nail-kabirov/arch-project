package postgres

import (
	"database/sql"
	"time"

	"github.com/pkg/errors"

	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/lot/app"
)

func NewBidRepository(client postgres.Client) app.BidRepository {
	return &bidRepository{client: client}
}

type bidRepository struct {
	client postgres.Client
}

func (repo *bidRepository) TryFindLastByLotID(lotID app.LotID) (*app.Bid, error) {
	const query = `
			SELECT lot_id, user_id, amount, created_at FROM bid WHERE lot_id = $1
			ORDER BY amount DESC
			LIMIT 1
		`

	var bid sqlxBid
	err := repo.client.Get(&bid, query, string(lotID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.WithStack(err)
	}
	res := sqlxBidToBid(&bid)
	return &res, nil
}

func (repo *bidRepository) Store(bid *app.Bid) error {
	const query = `
			INSERT INTO bid (lot_id, user_id, amount, created_at)
			VALUES (:lot_id, :user_id, :amount, :created_at)
		`

	bidx := sqlxBid{
		LotID:        string(bid.LotID),
		UserID:       string(bid.UserID),
		Amount:       bid.Amount.RawValue(),
		CreationTime: bid.CreationTime,
	}

	_, err := repo.client.NamedExec(query, &bidx)
	return errors.WithStack(err)
}

func sqlxBidToBid(bid *sqlxBid) app.Bid {
	return app.Bid{
		LotID:        app.LotID(bid.LotID),
		UserID:       app.UserID(bid.UserID),
		Amount:       app.AmountFromRawValue(bid.Amount),
		CreationTime: bid.CreationTime,
	}
}

type sqlxBid struct {
	LotID        string    `db:"lot_id"`
	UserID       string    `db:"user_id"`
	Amount       uint64    `db:"amount"`
	CreationTime time.Time `db:"created_at"`
}
