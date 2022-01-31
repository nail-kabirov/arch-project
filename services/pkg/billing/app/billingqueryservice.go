package app

func NewBillingQueryService(repoRead UserAccountEventRepository) BillingQueryService {
	return &billingQueryService{
		repoRead: repoRead,
	}
}

type QueryAccountStatus struct {
	Amount        Amount
	BlockedAmount Amount
}

type BillingQueryService interface {
	AccountBalance(userID UserID) (QueryAccountStatus, error)
}

type billingQueryService struct {
	repoRead UserAccountEventRepository
}

func (s *billingQueryService) AccountBalance(userID UserID) (QueryAccountStatus, error) {
	accountEvents, err := s.repoRead.FindAllByUserID(userID)
	if err != nil {
		return QueryAccountStatus{}, err
	}
	state := NewEmptyUserAccountState(userID)
	err = state.LoadEvents(accountEvents)
	if err != nil {
		return QueryAccountStatus{}, err
	}
	amount := state.Amount()
	blockedAmount := state.BlockedAmount()
	if amount == nil || blockedAmount == nil {
		amount = AmountFromRawValue(0)
		blockedAmount = AmountFromRawValue(0)
	}

	return QueryAccountStatus{
		Amount:        amount,
		BlockedAmount: blockedAmount,
	}, nil
}
