package app

import "github.com/pkg/errors"

var ErrUserAccountNotFound = errors.New("user account not found")
var ErrAccountAlreadyCreated = errors.New("user account already created")
var ErrEmptyPayment = errors.New("payment amount can not be equal to 0")
var ErrBlockPayment = errors.New("not enough funds for payment")
var ErrLotPaymentAlreadyBlocked = errors.New("payment for lot already blocked")
var ErrLotIDNotSpecified = errors.New("lot id absent in event")
var ErrUnblockPayment = errors.New("can't find matched blocked payment")
var ErrFinishPayment = errors.New("can't find blocked payment to finish it")

func NewEmptyUserAccountState(userID UserID) UserAccountState {
	return &userAccountState{
		userID:              userID,
		lotBlockedAmountMap: make(map[LotID]Amount),
	}
}

type UserAccountState interface {
	Amount() Amount
	BlockedAmount() Amount
	AddedEvents() []UserAccountEvent

	LoadEvents(events []UserAccountEvent) error

	AddCreateAccountEvent() error
	AddTopUpAccountEvent(amount Amount) error
	AddBlockPaymentEvent(lotID LotID, amount Amount) error
	AddUnblockPaymentEvent(lotID LotID, amount Amount) error
	AddFinishPaymentEvent(lotID LotID, amount Amount) error
	AddReceivePaymentEvent(lotID LotID, amount Amount) error
}

type userAccountState struct {
	userID              UserID
	totalAmount         Amount
	blockedAmount       Amount
	lotBlockedAmountMap map[LotID]Amount
	addedEvents         []UserAccountEvent
}

func (state *userAccountState) Amount() Amount {
	if state.totalAmount == nil || state.blockedAmount == nil {
		return nil
	}
	return AmountFromRawValue(state.totalAmount.RawValue() - state.blockedAmount.RawValue())
}

func (state *userAccountState) BlockedAmount() Amount {
	return state.blockedAmount
}

func (state *userAccountState) AddedEvents() []UserAccountEvent {
	return state.addedEvents
}

func (state *userAccountState) LoadEvents(events []UserAccountEvent) error {
	for _, event := range events {
		err := state.applyEvent(event)
		if err != nil {
			return err
		}
	}
	return nil
}

func (state *userAccountState) AddCreateAccountEvent() error {
	event := UserAccountEvent{
		UserID:    state.userID,
		EventType: createAccountEventType,
		Amount:    AmountFromRawValue(0),
	}
	return state.addEvent(event)
}

func (state *userAccountState) AddTopUpAccountEvent(amount Amount) error {
	event := UserAccountEvent{
		UserID:    state.userID,
		EventType: topUpAccountEventType,
		Amount:    amount,
	}
	return state.addEvent(event)
}

func (state *userAccountState) AddBlockPaymentEvent(lotID LotID, amount Amount) error {
	event := UserAccountEvent{
		UserID:    state.userID,
		EventType: blockPaymentEventType,
		LotID:     &lotID,
		Amount:    amount,
	}
	return state.addEvent(event)
}

func (state *userAccountState) AddUnblockPaymentEvent(lotID LotID, amount Amount) error {
	event := UserAccountEvent{
		UserID:    state.userID,
		EventType: unblockPaymentEventType,
		LotID:     &lotID,
		Amount:    amount,
	}
	return state.addEvent(event)
}

func (state *userAccountState) AddFinishPaymentEvent(lotID LotID, amount Amount) error {
	event := UserAccountEvent{
		UserID:    state.userID,
		EventType: finishPaymentEventType,
		LotID:     &lotID,
		Amount:    amount,
	}
	return state.addEvent(event)
}

func (state *userAccountState) AddReceivePaymentEvent(lotID LotID, amount Amount) error {
	event := UserAccountEvent{
		UserID:    state.userID,
		EventType: receivePaymentEventType,
		LotID:     &lotID,
		Amount:    amount,
	}
	return state.addEvent(event)
}

func (state *userAccountState) addEvent(event UserAccountEvent) error {
	err := state.applyEvent(event)
	if err != nil {
		return err
	}
	state.addedEvents = append(state.addedEvents, event)
	return nil
}

func (state *userAccountState) applyEvent(event UserAccountEvent) error {
	amount := event.Amount

	switch event.EventType {
	case createAccountEventType:
		return state.applyCreateAccountEvent()
	case topUpAccountEventType:
		return state.applyTopUpAccountEvent(amount)
	case blockPaymentEventType:
		if event.LotID == nil {
			return errors.WithStack(ErrLotIDNotSpecified)
		}
		return state.applyBlockPaymentEvent(*event.LotID, amount)
	case unblockPaymentEventType:
		if event.LotID == nil {
			return errors.WithStack(ErrLotIDNotSpecified)
		}
		return state.applyUnblockPaymentEvent(*event.LotID, amount)
	case finishPaymentEventType:
		if event.LotID == nil {
			return errors.WithStack(ErrLotIDNotSpecified)
		}
		return state.applyFinishPaymentEvent(*event.LotID, amount)
	case receivePaymentEventType:
		if event.LotID == nil {
			return errors.WithStack(ErrLotIDNotSpecified)
		}
		return state.applyReceivePaymentEvent(*event.LotID, amount)
	default:
		return errors.WithStack(errors.Errorf("unknown event type - '%s'", event.EventType))
	}
}

func (state *userAccountState) applyCreateAccountEvent() error {
	if state.totalAmount != nil || state.blockedAmount != nil {
		return errors.WithStack(ErrAccountAlreadyCreated)
	}
	state.totalAmount = AmountFromRawValue(0)
	state.blockedAmount = AmountFromRawValue(0)
	return nil
}

func (state *userAccountState) applyTopUpAccountEvent(amount Amount) error {
	if state.totalAmount == nil || state.blockedAmount == nil {
		return errors.WithStack(ErrUserAccountNotFound)
	}
	state.totalAmount = AmountFromRawValue(state.totalAmount.RawValue() + amount.RawValue())
	return nil
}

func (state *userAccountState) applyBlockPaymentEvent(lotID LotID, amount Amount) error {
	if state.totalAmount == nil || state.blockedAmount == nil {
		return errors.WithStack(ErrUserAccountNotFound)
	}
	if amount.RawValue() > state.Amount().RawValue() {
		return errors.WithStack(ErrBlockPayment)
	}
	if amount.RawValue() == 0 {
		return errors.WithStack(ErrEmptyPayment)
	}
	lotBlockedAmount, ok := state.lotBlockedAmountMap[lotID]
	if ok && lotBlockedAmount.RawValue() != 0 {
		return errors.WithStack(ErrLotPaymentAlreadyBlocked)
	}
	state.blockedAmount = AmountFromRawValue(state.blockedAmount.RawValue() + amount.RawValue())
	state.lotBlockedAmountMap[lotID] = amount
	return nil
}

func (state *userAccountState) applyUnblockPaymentEvent(lotID LotID, amount Amount) error {
	if state.totalAmount == nil || state.blockedAmount == nil {
		return errors.WithStack(ErrUserAccountNotFound)
	}
	lotBlockedAmount, ok := state.lotBlockedAmountMap[lotID]
	if !ok || lotBlockedAmount.RawValue() != amount.RawValue() || state.blockedAmount.RawValue() < amount.RawValue() {
		return errors.WithStack(ErrUnblockPayment)
	}
	state.blockedAmount = AmountFromRawValue(state.blockedAmount.RawValue() - amount.RawValue())
	delete(state.lotBlockedAmountMap, lotID)
	return nil
}

func (state *userAccountState) applyFinishPaymentEvent(lotID LotID, amount Amount) error {
	if state.totalAmount == nil || state.blockedAmount == nil {
		return errors.WithStack(ErrUserAccountNotFound)
	}
	lotBlockedAmount, ok := state.lotBlockedAmountMap[lotID]
	if !ok || lotBlockedAmount.RawValue() != amount.RawValue() || state.blockedAmount.RawValue() < amount.RawValue() {
		return errors.WithStack(ErrFinishPayment)
	}
	state.totalAmount = AmountFromRawValue(state.totalAmount.RawValue() - amount.RawValue())
	state.blockedAmount = AmountFromRawValue(state.blockedAmount.RawValue() - amount.RawValue())
	delete(state.lotBlockedAmountMap, lotID)
	return nil
}

func (state *userAccountState) applyReceivePaymentEvent(_ LotID, amount Amount) error {
	if state.totalAmount == nil || state.blockedAmount == nil {
		return errors.WithStack(ErrUserAccountNotFound)
	}
	state.totalAmount = AmountFromRawValue(state.totalAmount.RawValue() + amount.RawValue())
	return nil
}
