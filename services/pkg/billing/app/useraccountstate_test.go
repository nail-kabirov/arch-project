package app

import (
	"arch-homework/pkg/common/app/uuid"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"testing"
)

var testUserID = UserID(uuid.GenerateNew())
var testLotID = LotID(uuid.GenerateNew())
var emptyAmount = AmountFromRawValue(0)

func TestInitialState(t *testing.T) {
	state := NewEmptyUserAccountState(testUserID)
	assert.Nil(t, state.Amount())
	assert.Nil(t, state.BlockedAmount())
}

func TestCreateAccountEvent(t *testing.T) {
	state := NewEmptyUserAccountState(testUserID)
	assert.Nil(t, state.AddCreateAccountEvent())
	assert.Equal(t, emptyAmount, state.Amount())
	assert.Equal(t, emptyAmount, state.BlockedAmount())

	addedEvents := state.AddedEvents()
	assert.Len(t, addedEvents, 1)
	assert.Equal(t, UserAccountEvent{
		UserID:    testUserID,
		LotID:     nil,
		EventType: createAccountEventType,
		Amount:    emptyAmount,
	}, addedEvents[0])
}

func TestCreateAccountEventAfterAnotherFailed(t *testing.T) {
	state := NewEmptyUserAccountState(testUserID)
	assert.Nil(t, state.AddCreateAccountEvent())
	err := state.AddCreateAccountEvent()
	assert.Equal(t, ErrAccountAlreadyCreated, errors.Cause(err))
}

func TestAddEventsOnNotCreatedAccountFailed(t *testing.T) {
	state := NewEmptyUserAccountState(testUserID)

	amount := AmountFromRawValue(123)

	assert.Equal(t, ErrUserAccountNotFound, errors.Cause(state.AddTopUpAccountEvent(amount)))
	assert.Equal(t, ErrUserAccountNotFound, errors.Cause(state.AddBlockPaymentEvent(testLotID, amount)))
	assert.Equal(t, ErrUserAccountNotFound, errors.Cause(state.AddUnblockPaymentEvent(testLotID, amount)))
	assert.Equal(t, ErrUserAccountNotFound, errors.Cause(state.AddFinishPaymentEvent(testLotID, amount)))
	assert.Equal(t, ErrUserAccountNotFound, errors.Cause(state.AddReceivePaymentEvent(testLotID, amount)))
}

func TestTopUpEvent(t *testing.T) {
	state := createdOnlyState(t)
	amount := AmountFromRawValue(12345)
	assert.Nil(t, state.AddTopUpAccountEvent(amount))
	assert.Equal(t, amount, state.Amount())
	assert.Equal(t, emptyAmount, state.BlockedAmount())

	addedEvents := state.AddedEvents()
	assert.Len(t, addedEvents, 1)
	assert.Equal(t, UserAccountEvent{
		UserID:    testUserID,
		LotID:     nil,
		EventType: topUpAccountEventType,
		Amount:    amount,
	}, addedEvents[0])
}

func TestBlockPaymentEvent(t *testing.T) {
	state := createdOnlyState(t)
	totalAmount := AmountFromRawValue(12345)
	paymentAmount := AmountFromRawValue(10000)

	assert.Nil(t, state.AddTopUpAccountEvent(totalAmount))
	assert.Nil(t, state.AddBlockPaymentEvent(testLotID, paymentAmount))

	assert.Equal(t, AmountFromRawValue(2345), state.Amount())
	assert.Equal(t, paymentAmount, state.BlockedAmount())

	addedEvents := state.AddedEvents()
	assert.Len(t, addedEvents, 2)
	assert.Equal(t, topUpAccountEventType, addedEvents[0].EventType)
	assert.Equal(t, UserAccountEvent{
		UserID:    testUserID,
		LotID:     &testLotID,
		EventType: blockPaymentEventType,
		Amount:    paymentAmount,
	}, addedEvents[1])
}

func TestBlockPaymentWithoutEnoughFundsFailed(t *testing.T) {
	state := createdOnlyState(t)
	totalAmount := AmountFromRawValue(2345)
	paymentAmount := AmountFromRawValue(10000)

	assert.Nil(t, state.AddTopUpAccountEvent(totalAmount))
	err := state.AddBlockPaymentEvent(testLotID, paymentAmount)
	assert.Equal(t, ErrBlockPayment, errors.Cause(err))
}

func TestBlockEmptyPaymentFailed(t *testing.T) {
	state := createdOnlyState(t)
	totalAmount := AmountFromRawValue(2345)
	assert.Nil(t, state.AddTopUpAccountEvent(totalAmount))
	err := state.AddBlockPaymentEvent(testLotID, emptyAmount)
	assert.Equal(t, ErrEmptyPayment, errors.Cause(err))
}

func TestDuplicateBlockPaymentFailed(t *testing.T) {
	state := createdOnlyState(t)

	assert.Nil(t, state.AddTopUpAccountEvent(AmountFromRawValue(12345)))
	assert.Nil(t, state.AddBlockPaymentEvent(testLotID, AmountFromRawValue(1000)))
	err := state.AddBlockPaymentEvent(testLotID, AmountFromRawValue(1000))
	assert.Equal(t, ErrLotPaymentAlreadyBlocked, errors.Cause(err))
}

func TestUnblockPaymentEvent(t *testing.T) {
	state := createdOnlyState(t)
	totalAmount := AmountFromRawValue(12345)
	paymentAmount := AmountFromRawValue(10000)

	assert.Nil(t, state.AddTopUpAccountEvent(totalAmount))
	assert.Nil(t, state.AddBlockPaymentEvent(testLotID, paymentAmount))
	assert.Nil(t, state.AddUnblockPaymentEvent(testLotID, paymentAmount))

	assert.Equal(t, totalAmount, state.Amount())
	assert.Equal(t, emptyAmount, state.BlockedAmount())

	addedEvents := state.AddedEvents()
	assert.Len(t, addedEvents, 3)
	assert.Equal(t, topUpAccountEventType, addedEvents[0].EventType)
	assert.Equal(t, blockPaymentEventType, addedEvents[1].EventType)
	assert.Equal(t, UserAccountEvent{
		UserID:    testUserID,
		LotID:     &testLotID,
		EventType: unblockPaymentEventType,
		Amount:    paymentAmount,
	}, addedEvents[2])
}

func TestUnblockUnmatchedPaymentFailed(t *testing.T) {
	state := createdOnlyState(t)
	totalAmount := AmountFromRawValue(12345)
	paymentAmount := AmountFromRawValue(10000)

	assert.Nil(t, state.AddTopUpAccountEvent(totalAmount))
	assert.Nil(t, state.AddBlockPaymentEvent(testLotID, paymentAmount))

	err := state.AddUnblockPaymentEvent(testLotID, AmountFromRawValue(9000))
	assert.Equal(t, ErrUnblockPayment, errors.Cause(err))

	err = state.AddUnblockPaymentEvent(testLotID, AmountFromRawValue(10001))
	assert.Equal(t, ErrUnblockPayment, errors.Cause(err))
}

func TestFinishPaymentEvent(t *testing.T) {
	state := createdOnlyState(t)
	totalAmount := AmountFromRawValue(12345)
	paymentAmount := AmountFromRawValue(10000)

	assert.Nil(t, state.AddTopUpAccountEvent(totalAmount))
	assert.Nil(t, state.AddBlockPaymentEvent(testLotID, paymentAmount))
	assert.Nil(t, state.AddFinishPaymentEvent(testLotID, paymentAmount))

	assert.Equal(t, AmountFromRawValue(2345), state.Amount())
	assert.Equal(t, emptyAmount, state.BlockedAmount())

	addedEvents := state.AddedEvents()
	assert.Len(t, addedEvents, 3)
	assert.Equal(t, topUpAccountEventType, addedEvents[0].EventType)
	assert.Equal(t, blockPaymentEventType, addedEvents[1].EventType)
	assert.Equal(t, UserAccountEvent{
		UserID:    testUserID,
		LotID:     &testLotID,
		EventType: finishPaymentEventType,
		Amount:    paymentAmount,
	}, addedEvents[2])
}

func TestFinishUnmatchedPaymentFailed(t *testing.T) {
	state := createdOnlyState(t)
	totalAmount := AmountFromRawValue(12345)
	paymentAmount := AmountFromRawValue(10000)

	assert.Nil(t, state.AddTopUpAccountEvent(totalAmount))
	assert.Nil(t, state.AddBlockPaymentEvent(testLotID, paymentAmount))

	err := state.AddFinishPaymentEvent(testLotID, AmountFromRawValue(9000))
	assert.Equal(t, ErrFinishPayment, errors.Cause(err))

	err = state.AddFinishPaymentEvent(testLotID, AmountFromRawValue(10001))
	assert.Equal(t, ErrFinishPayment, errors.Cause(err))
}

func TestPaymentReceivedEvent(t *testing.T) {
	state := createdOnlyState(t)
	paymentAmount := AmountFromRawValue(1234)

	assert.Nil(t, state.AddReceivePaymentEvent(testLotID, paymentAmount))

	assert.Equal(t, paymentAmount, state.Amount())
	assert.Equal(t, emptyAmount, state.BlockedAmount())

	addedEvents := state.AddedEvents()
	assert.Len(t, addedEvents, 1)
	assert.Equal(t, UserAccountEvent{
		UserID:    testUserID,
		LotID:     &testLotID,
		EventType: receivePaymentEventType,
		Amount:    paymentAmount,
	}, addedEvents[0])
}

func createdOnlyState(t *testing.T) UserAccountState {
	state := NewEmptyUserAccountState(testUserID)
	assert.Nil(t, state.LoadEvents([]UserAccountEvent{{
		UserID:    testUserID,
		LotID:     nil,
		EventType: createAccountEventType,
		Amount:    emptyAmount,
	}}))
	assert.Empty(t, state.AddedEvents())
	return state
}
