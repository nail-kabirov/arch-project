package app

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/uuid"
	"encoding/json"
)

const typeLotSent = "delivery.lot_sent"
const typeLotReceived = "delivery.lot_received"

func NewLotSentEvent(lotID LotID) integrationevent.EventData {
	body, _ := json.Marshal(lotDeliveryEventBody{
		LotID: string(lotID),
	})

	return integrationevent.EventData{
		UID:  newUID(),
		Type: typeLotSent,
		Body: string(body),
	}
}

func NewLotReceivedEvent(lotID LotID) integrationevent.EventData {
	body, _ := json.Marshal(lotDeliveryEventBody{
		LotID: string(lotID),
	})

	return integrationevent.EventData{
		UID:  newUID(),
		Type: typeLotReceived,
		Body: string(body),
	}
}

func newUID() integrationevent.EventUID {
	return integrationevent.EventUID(uuid.GenerateNew())
}

type lotDeliveryEventBody struct {
	LotID string `json:"lot_id"`
}
