package app

import (
	"arch-homework/pkg/common/app/integrationevent"
	"arch-homework/pkg/common/app/uuid"
	"encoding/json"
)

const typeLotWon = "lot.lot_won"
const typeLotClosed = "lot.lot_closed"
const typeLotSent = "lot.lot_sent"
const typeLotReceived = "lot.lot_received"
const typeBidOutbid = "lot.bid_outbid"
const typeBidCancelled = "lot.bid_cancelled"

func NewLotWonEvent(lotID LotID, userID, lotOwnerID UserID) integrationevent.EventData {
	body, _ := json.Marshal(lotWonEventBody{
		LotID:      string(lotID),
		UserID:     string(userID),
		LotOwnerID: string(lotOwnerID),
	})

	return integrationevent.EventData{
		UID:  newUID(),
		Type: typeLotWon,
		Body: string(body),
	}
}

func NewLotClosedEvent(lotID LotID, lotOwnerID UserID) integrationevent.EventData {
	body, _ := json.Marshal(lotClosedEventBody{
		LotID:      string(lotID),
		LotOwnerID: string(lotOwnerID),
	})

	return integrationevent.EventData{
		UID:  newUID(),
		Type: typeLotClosed,
		Body: string(body),
	}
}

func NewLotSentEvent(lotID LotID, userID, lotOwnerID UserID) integrationevent.EventData {
	body, _ := json.Marshal(lotSentEventBody{
		LotID:      string(lotID),
		UserID:     string(userID),
		LotOwnerID: string(lotOwnerID),
	})

	return integrationevent.EventData{
		UID:  newUID(),
		Type: typeLotSent,
		Body: string(body),
	}
}

func NewLotReceivedEvent(lotID LotID, userID, lotOwnerID UserID, finalAmount Amount) integrationevent.EventData {
	body, _ := json.Marshal(lotReceivedEventBody{
		LotID:       string(lotID),
		UserID:      string(userID),
		LotOwnerID:  string(lotOwnerID),
		FinalAmount: finalAmount.RawValue(),
	})

	return integrationevent.EventData{
		UID:  newUID(),
		Type: typeLotReceived,
		Body: string(body),
	}
}

func NewBidOutbidEvent(lotID LotID, userID UserID, bidAmount Amount) integrationevent.EventData {
	body, _ := json.Marshal(bidEventBody{
		UserID:    string(userID),
		LotID:     string(lotID),
		BidAmount: bidAmount.RawValue(),
	})

	return integrationevent.EventData{
		UID:  newUID(),
		Type: typeBidOutbid,
		Body: string(body),
	}
}

func NewBidCancelledEvent(lotID LotID, userID UserID, bidAmount Amount) integrationevent.EventData {
	body, _ := json.Marshal(bidEventBody{
		UserID:    string(userID),
		LotID:     string(lotID),
		BidAmount: bidAmount.RawValue(),
	})

	return integrationevent.EventData{
		UID:  newUID(),
		Type: typeBidCancelled,
		Body: string(body),
	}
}

func newUID() integrationevent.EventUID {
	return integrationevent.EventUID(uuid.GenerateNew())
}

type lotWonEventBody struct {
	LotID      string `json:"lot_id"`
	UserID     string `json:"user_id"`
	LotOwnerID string `json:"lot_owner_id"`
}

type lotClosedEventBody struct {
	LotID      string `json:"lot_id"`
	LotOwnerID string `json:"lot_owner_id"`
}

type lotSentEventBody struct {
	LotID      string `json:"lot_id"`
	UserID     string `json:"user_id"`
	LotOwnerID string `json:"lot_owner_id"`
}

type lotReceivedEventBody struct {
	LotID       string `json:"lot_id"`
	UserID      string `json:"user_id"`
	LotOwnerID  string `json:"lot_owner_id"`
	FinalAmount uint64 `json:"final_amount"`
}

type bidEventBody struct {
	UserID    string `json:"user_id"`
	LotID     string `json:"lot_id"`
	BidAmount uint64 `json:"bid_amount"`
}
