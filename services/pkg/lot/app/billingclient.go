package app

type BillingClient interface {
	ProcessOrderPayment(userID UserID, lotID LotID, price Amount) (succeeded bool, err error)
}
