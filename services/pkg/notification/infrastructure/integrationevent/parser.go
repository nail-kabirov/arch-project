package integrationevent

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/notification/app"

	"encoding/json"

	"github.com/pkg/errors"
)

const typeLotWon = "lot.lot_won"
const typeLotClosed = "lot.lot_closed"
const typeLotSent = "lot.lot_sent"
const typeLotReceived = "lot.lot_received"
const typeBidOutbid = "lot.bid_outbid"

func NewEventParser() app.IntegrationEventParser {
	return eventParser{}
}

type eventParser struct {
}

func (e eventParser) ParseIntegrationEvent(event integrationevent.EventData) (app.HandledEvent, error) {
	switch event.Type {
	case typeLotWon:
		return parseLotWonEvent(event.Body)
	case typeLotClosed:
		return parseLotClosedEvent(event.Body)
	case typeLotSent:
		return parseLotSentEvent(event.Body)
	case typeLotReceived:
		return parseLotReceivedEvent(event.Body)
	case typeBidOutbid:
		return parseBidOutbidEvent(event.Body)
	default:
		return nil, nil
	}
}

func parseLotWonEvent(strBody string) (app.HandledEvent, error) {
	body, err := parseLotEvent(strBody)
	if err != nil {
		return nil, err
	}
	return app.NewLotWonEvent(app.LotID(body.LotID), app.UserID(body.LotOwnerID), app.UserID(body.UserID)), nil
}

func parseLotClosedEvent(strBody string) (app.HandledEvent, error) {
	body, err := parseLotEvent(strBody)
	if err != nil {
		return nil, err
	}
	return app.NewLotClosedEvent(app.LotID(body.LotID), app.UserID(body.LotOwnerID)), nil
}

func parseLotSentEvent(strBody string) (app.HandledEvent, error) {
	body, err := parseLotEvent(strBody)
	if err != nil {
		return nil, err
	}
	return app.NewLotSentEvent(app.LotID(body.LotID), app.UserID(body.LotOwnerID), app.UserID(body.UserID)), nil
}

func parseLotReceivedEvent(strBody string) (app.HandledEvent, error) {
	body, err := parseLotEvent(strBody)
	if err != nil {
		return nil, err
	}
	return app.NewLotReceivedEvent(app.LotID(body.LotID), app.UserID(body.LotOwnerID), app.UserID(body.UserID)), nil
}

func parseBidOutbidEvent(strBody string) (app.HandledEvent, error) {
	var body bidEventBody
	err := json.Unmarshal([]byte(strBody), &body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.LotID)
	if err != nil {
		return body, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.UserID)
	if err != nil {
		return body, errors.WithStack(err)
	}
	return app.NewBidOutbidEvent(app.LotID(body.LotID), app.UserID(body.UserID)), nil
}

func parseLotEvent(strBody string) (lotEventBody, error) {
	var body lotEventBody
	err := json.Unmarshal([]byte(strBody), &body)
	if err != nil {
		return body, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.LotID)
	if err != nil {
		return body, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.LotOwnerID)
	if err != nil {
		return body, errors.WithStack(err)
	}
	if body.UserID != "" {
		err = uuid.ValidateUUID(body.UserID)
	}
	return body, errors.WithStack(err)
}

type lotEventBody struct {
	LotID      string `json:"lot_id"`
	UserID     string `json:"user_id,omitempty"`
	LotOwnerID string `json:"lot_owner_id"`
}

type bidEventBody struct {
	LotID  string `json:"lot_id"`
	UserID string `json:"user_id"`
}
