package postgres

import (
	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/lot/app"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"database/sql"
	"strings"
	"time"
)

func NewLotQueryService(client postgres.Client, userClient app.UserClient) app.LotQueryService {
	return &lotQueryService{
		client:     client,
		userClient: userClient,
	}
}

type lotQueryService struct {
	client     postgres.Client
	userClient app.UserClient
}

func (s *lotQueryService) Get(lotID app.LotID) (*app.LotQueryData, error) {
	const sqlQuery = `
			SELECT l.id,
				   l.owner_id,
				   l.description,
				   l.status,
				   l.start_price,
				   l.buy_it_now_price,
				   l.end_time,
				   l.created_at,
				   b.user_id AS last_bidder_id,
				   b.amount  AS last_bid_amount
			FROM lot AS l
					 LEFT JOIN LATERAL (SELECT user_id, amount FROM bid WHERE lot_id = l.id ORDER BY amount DESC LIMIT 1) AS b ON TRUE
			WHERE l.id = $1
		`

	var lot sqlxLotQueryData
	err := s.client.Get(&lot, sqlQuery, string(lotID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.WithStack(app.ErrLotNotFound)
		}
		return nil, errors.WithStack(err)
	}
	res, err := s.toLotQueryData(&lot)
	return &res, err
}

func (s *lotQueryService) FindAvailable(userID app.UserID, createdAfter *time.Time, searchString *string, withParticipationOnly bool, wonOnly bool) ([]app.LotQueryData, error) {
	const sqlQuery = `
			SELECT l.id,
				   l.owner_id,
				   l.description,
				   l.status,
				   l.start_price,
				   l.buy_it_now_price,
				   l.end_time,
				   l.created_at,
				   b.user_id AS last_bidder_id,
				   b.amount  AS last_bid_amount
			FROM lot AS l
					 LEFT JOIN LATERAL (SELECT user_id, amount FROM bid WHERE lot_id = l.id ORDER BY amount DESC LIMIT 1) AS b ON TRUE 
		`

	query := sqlQuery
	var params []interface{}
	var conditions []string

	if withParticipationOnly {
		query += " INNER JOIN LATERAL(SELECT DISTINCT(user_id) AS user_id FROM bid WHERE lot_id = l.id) AS b2 ON b2.user_id = ?"
		params = append(params, string(userID))
	} else {
		// check status
		if wonOnly {
			conditions = append(conditions, "l.status <> ?")
			params = append(params, string(app.LotStatusActive))
		} else {
			conditions = append(conditions, "l.status = ?")
			params = append(params, string(app.LotStatusActive))
		}
	}

	if wonOnly {
		conditions = append(conditions, "b.user_id = ?")
		params = append(params, string(userID))
	}
	if createdAfter != nil {
		conditions = append(conditions, "l.created_at > ?")
		params = append(params, *createdAfter)
	}
	if searchString != nil {
		searchStr := "%" + strings.ReplaceAll(*searchString, "%", "\\%") + "%"
		conditions = append(conditions, "l.description LIKE ?")
		params = append(params, searchStr)
	}

	conditions = append(conditions, "l.owner_id <> ?")
	params = append(params, string(userID))

	query += " WHERE " + strings.Join(conditions, " AND ")
	query = sqlx.Rebind(sqlx.DOLLAR, query)

	var lots []*sqlxLotQueryData
	err := s.client.Select(&lots, query, params...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	lotsData := make([]app.LotQueryData, 0, len(lots))
	for _, lot := range lots {
		lotData, err := s.toLotQueryData(lot)
		if err != nil {
			return nil, err
		}
		lotsData = append(lotsData, lotData)
	}
	return lotsData, nil
}

func (s *lotQueryService) FindByOwnerID(ownerID app.UserID) ([]app.LotWithBidsQueryData, error) {
	const sqlQuery = `
			SELECT id, owner_id, description, status, start_price, buy_it_now_price, end_time, created_at FROM lot
			WHERE owner_id = $1
		`

	var lots []*sqlxLot
	err := s.client.Select(&lots, sqlQuery, string(ownerID))
	if err != nil || len(lots) == 0 {
		return nil, errors.WithStack(err)
	}

	lotIDs := make([]string, 0, len(lots))
	for _, lot := range lots {
		lotIDs = append(lotIDs, lot.ID)
	}
	lotBidsMap, err := s.lotBidsMap(lotIDs)
	if err != nil {
		return nil, err
	}

	res := make([]app.LotWithBidsQueryData, 0, len(lots))
	for _, lot := range lots {
		lotWithBids := app.LotWithBidsQueryData{Lot: sqlxLotToLot(lot)}
		if bids, ok := lotBidsMap[lot.ID]; ok {
			lotWithBids.Bids = bids
		}
		res = append(res, lotWithBids)
	}
	return res, nil
}

func (s *lotQueryService) lotBidsMap(lotIDs []string) (map[string][]app.BidQueryData, error) {
	const sqlQuery = `SELECT lot_id, user_id, amount, created_at FROM bid WHERE lot_id IN (?)`

	query, params, err := sqlx.In(sqlQuery, lotIDs)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	query = sqlx.Rebind(sqlx.DOLLAR, query)

	var bids []*sqlxBid
	err = s.client.Select(&bids, query, params...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	res := make(map[string][]app.BidQueryData)
	for _, bid := range bids {
		userLogin, err := s.userClient.GetUserLogin(app.UserID(bid.UserID))
		if err != nil {
			return nil, err
		}

		res[bid.LotID] = append(res[bid.LotID], app.BidQueryData{
			Bid:       sqlxBidToBid(bid),
			UserLogin: userLogin,
		})
	}
	return res, nil
}

func (s *lotQueryService) toLotQueryData(lot *sqlxLotQueryData) (app.LotQueryData, error) {
	ownerLogin, err := s.userClient.GetUserLogin(app.UserID(lot.OwnerID))
	if err != nil {
		return app.LotQueryData{}, err
	}
	data := app.LotQueryData{
		Lot: app.Lot{
			ID:           app.LotID(lot.ID),
			OwnerID:      app.UserID(lot.OwnerID),
			Description:  lot.Description,
			StartPrice:   app.AmountFromRawValue(lot.StartPrice),
			Status:       app.LotStatus(lot.Status),
			EndTime:      lot.EndTime,
			CreationTime: lot.CreationTime,
		},
		OwnerLogin: ownerLogin,
	}
	if lot.BuyItNowPrice.Valid {
		price := app.AmountFromRawValue(uint64(lot.BuyItNowPrice.Int64))
		data.BuyItNowPrice = &price
	}
	if lot.LastBidAmount.Valid {
		amount := app.AmountFromRawValue(uint64(lot.LastBidAmount.Int64))
		data.LastBidAmount = &amount
	}
	if lot.LastBidderID.Valid {
		lastBidderID := app.UserID(lot.LastBidderID.String)
		data.LastBidderID = &lastBidderID
	}
	return data, nil
}

type sqlxLotQueryData struct {
	ID            string         `db:"id"`
	OwnerID       string         `db:"owner_id"`
	Description   string         `db:"description"`
	Status        string         `db:"status"`
	StartPrice    uint64         `db:"start_price"`
	BuyItNowPrice sql.NullInt64  `db:"buy_it_now_price"`
	EndTime       time.Time      `db:"end_time"`
	CreationTime  time.Time      `db:"created_at"`
	LastBidderID  sql.NullString `db:"last_bidder_id"`
	LastBidAmount sql.NullInt64  `db:"last_bid_amount"`
}
