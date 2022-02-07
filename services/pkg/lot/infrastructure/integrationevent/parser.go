package integrationevent

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/lot/app"

	"encoding/json"

	"github.com/pkg/errors"
)

const typeDeliveryLotSent = "delivery.lot_sent"
const typeDeliveryLotReceived = "delivery.lot_received"

func NewEventParser() app.IntegrationEventParser {
	return eventParser{}
}

type eventParser struct {
}

func (e eventParser) ParseIntegrationEvent(event integrationevent.EventData) (app.HandledEvent, error) {
	switch event.Type {
	case typeDeliveryLotSent:
		return parseLotSentEvent(event.Body)
	case typeDeliveryLotReceived:
		return parseLotReceivedEvent(event.Body)
	default:
		return nil, nil
	}
}

func parseLotSentEvent(strBody string) (app.HandledEvent, error) {
	var body deliveryLotEventBody
	err := json.Unmarshal([]byte(strBody), &body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.LotID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return app.NewDeliveryLotSentEvent(app.LotID(body.LotID)), nil
}

func parseLotReceivedEvent(strBody string) (app.HandledEvent, error) {
	var body deliveryLotEventBody
	err := json.Unmarshal([]byte(strBody), &body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	err = uuid.ValidateUUID(body.LotID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return app.NewDeliveryLotReceivedEvent(app.LotID(body.LotID)), nil
}

type deliveryLotEventBody struct {
	LotID string `json:"lot_id"`
}
