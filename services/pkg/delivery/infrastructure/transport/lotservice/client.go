package lotservice

import (
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/infrastructure/httpclient"
	"arch-homework/pkg/delivery/app"

	"github.com/pkg/errors"

	"fmt"
	"net/http"
)

const lotInfoURLTpl = "/internal/api/v1/lot/%s"

func NewClient(client http.Client, serviceHost string) app.LotServiceClient {
	return &lotServiceClient{httpClient: httpclient.NewClient(client, serviceHost)}
}

type lotServiceClient struct {
	httpClient httpclient.Client
}

func (c *lotServiceClient) FindFinishedLotInfo(id app.LotID) (*app.LotInfo, error) {
	requestURL := fmt.Sprintf(lotInfoURLTpl, string(id))
	response := lotInfoResponse{}
	err := c.httpClient.MakeJSONRequest(nil, &response, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	if response.Status != string(app.LotStatusFinished) {
		return nil, errors.WithStack(app.ErrInvalidLotStatus)
	}

	if err = uuid.ValidateUUID(response.ID); err != nil {
		return nil, errors.WithStack(err)
	}
	if err = uuid.ValidateUUID(response.OwnerID); err != nil {
		return nil, errors.WithStack(err)
	}
	if err = uuid.ValidateUUID(response.LastBidderID); err != nil {
		return nil, errors.WithStack(err)
	}

	info := app.LotInfo{
		ID:         app.LotID(response.ID),
		OwnerID:    app.UserID(response.OwnerID),
		ReceiverID: app.UserID(response.LastBidderID),
	}

	return &info, nil
}

type lotInfoResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	OwnerID      string `json:"ownerId"`
	LastBidderID string `json:"lastBidderID"`
}
