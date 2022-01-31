package integrationevent

import (
	"arch-homework/pkg/billing/app"
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/uuid"

	"encoding/json"

	"github.com/pkg/errors"
)

const typeUserRegistered = "user.user_registered"
const typeLotBidOutbid = "lot.bid_outbid"
const typeLotBidCancelled = "lot.bid_cancelled"
const typeLotReceived = "lot.lot_received"

func NewEventParser() app.IntegrationEventParser {
	return eventParser{}
}

type eventParser struct {
}

func (e eventParser) ParseIntegrationEvent(event integrationevent.EventData) (app.UserEvent, error) {
	switch event.Type {
	case typeUserRegistered:
		return parseUserRegisteredEvent(event.Body)
	case typeLotBidOutbid:
		return parseLotBidOutbidEvent(event.Body)
	case typeLotBidCancelled:
		return parseLotBidCancelledEvent(event.Body)
	case typeLotReceived:
		return parseLotReceivedEvent(event.Body)
	default:
		return nil, nil
	}
}

func parseUserRegisteredEvent(strBody string) (app.UserEvent, error) {
	var body userRegisteredEventBody
	err := json.Unmarshal([]byte(strBody), &body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.UserID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return app.NewUserRegisteredEvent(app.UserID(body.UserID), body.Login), nil
}

func parseLotBidOutbidEvent(strBody string) (app.UserEvent, error) {
	var body lotBidOutbidEventBody
	err := json.Unmarshal([]byte(strBody), &body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.UserID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.LotID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return app.NewLotBidOutbidEvent(app.UserID(body.UserID), app.LotID(body.LotID), app.AmountFromRawValue(body.BidAmount)), nil
}

func parseLotBidCancelledEvent(strBody string) (app.UserEvent, error) {
	var body lotBidCancelledEventBody
	err := json.Unmarshal([]byte(strBody), &body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.UserID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.LotID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return app.NewLotBidCancelledEvent(app.UserID(body.UserID), app.LotID(body.LotID), app.AmountFromRawValue(body.BidAmount)), nil
}

func parseLotReceivedEvent(strBody string) (app.UserEvent, error) {
	var body lotReceivedEventBody
	err := json.Unmarshal([]byte(strBody), &body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.UserID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.LotID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.LotOwnerID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return app.NewLotReceivedEvent(app.UserID(body.UserID), app.LotID(body.LotID), app.UserID(body.LotOwnerID), app.AmountFromRawValue(body.FinalAmount)), nil
}

type userRegisteredEventBody struct {
	UserID string `json:"user_id"`
	Login  string `json:"login"`
}

type lotBidOutbidEventBody struct {
	UserID    string `json:"user_id"`
	LotID     string `json:"lot_id"`
	BidAmount uint64 `json:"bid_amount"`
}

type lotBidCancelledEventBody struct {
	UserID    string `json:"user_id"`
	LotID     string `json:"lot_id"`
	BidAmount uint64 `json:"bid_amount"`
}

type lotReceivedEventBody struct {
	UserID      string `json:"user_id"`
	LotID       string `json:"lot_id"`
	LotOwnerID  string `json:"lot_owner_id"`
	FinalAmount uint64 `json:"final_amount"`
}
