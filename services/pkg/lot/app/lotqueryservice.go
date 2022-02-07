package app

type LotQueryData struct {
	Lot
	OwnerLogin    string
	LastBidAmount *Amount
	LastBidderID  *UserID
}

type BidQueryData struct {
	Bid
	UserLogin string
}

type LotWithBidsQueryData struct {
	Lot
	Bids []BidQueryData
}

type LotQueryService interface {
	Get(lotID LotID) (*LotQueryData, error)
	FindAvailable(userID UserID, searchString *string, withParticipationOnly bool, wonOnly bool) ([]LotQueryData, error)
	FindByOwnerID(ownerID UserID) ([]LotWithBidsQueryData, error)
}
