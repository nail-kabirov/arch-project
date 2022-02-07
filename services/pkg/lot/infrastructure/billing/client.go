package billing

import (
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/infrastructure/httpclient"
	"arch-homework/pkg/lot/app"

	"github.com/pkg/errors"

	"net/http"
	"time"
)

const processPaymentURL = "/internal/api/v1/payment"
const maxAttemptCount = 10

func NewClient(client http.Client, serviceHost string) app.BillingClient {
	return &billingClient{httpClient: httpclient.NewClient(client, serviceHost)}
}

type billingClient struct {
	httpClient httpclient.Client
}

func (c *billingClient) ProcessOrderPayment(userID app.UserID, lotID app.LotID, price app.Amount) (succeeded bool, err error) {
	requestID := string(uuid.GenerateNew())

	request := processPaymentRequest{
		UserID: string(userID),
		LotID:  string(lotID),
		Amount: price.Value(),
	}

	for i := 0; i < maxAttemptCount; i++ {
		err = c.httpClient.MakeJSONRequest(request, nil, http.MethodPost, processPaymentURL, &requestID)
		if err == nil {
			return true, nil
		}
		if e, ok := errors.Cause(err).(*httpclient.HTTPError); ok {
			if e.StatusCode == http.StatusBadRequest {
				return false, nil
			}
			if i > 0 && e.StatusCode == http.StatusConflict {
				// request already processed
				return true, nil
			}
		}

		// wait before next attempt
		time.Sleep(time.Millisecond * 100)
	}

	return false, err
}

type processPaymentRequest struct {
	UserID string  `json:"userID"`
	LotID  string  `json:"lotID"`
	Amount float64 `json:"amount"`
}
