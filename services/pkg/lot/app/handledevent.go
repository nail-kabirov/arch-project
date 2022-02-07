package app

type HandledEvent interface {
}

func NewDeliveryLotSentEvent(lotID LotID) HandledEvent {
	return deliveryLotSentEvent{
		lotID: lotID,
	}
}

func NewDeliveryLotReceivedEvent(lotID LotID) HandledEvent {
	return deliveryLotReceivedEvent{
		lotID: lotID,
	}
}

type deliveryLotSentEvent struct {
	lotID LotID
}

type deliveryLotReceivedEvent struct {
	lotID LotID
}
