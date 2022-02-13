package userservice

import (
	"arch-homework/pkg/common/infrastructure/httpclient"
	"arch-homework/pkg/delivery/app"

	"fmt"
	"net/http"
)

const userProfileURLTpl = "/internal/api/v1/user/%s/profile"

func NewClient(client http.Client, serviceHost string) app.UserServiceClient {
	return &userClient{
		httpClient: httpclient.NewClient(client, serviceHost),
	}
}

type userClient struct {
	httpClient httpclient.Client
}

func (c *userClient) GetUserInfo(userID app.UserID) (app.UserInfo, error) {
	requestURL := fmt.Sprintf(userProfileURLTpl, string(userID))
	var response userProfileResponse
	err := c.httpClient.MakeJSONRequest(nil, &response, http.MethodGet, requestURL, nil)
	if err != nil {
		return app.UserInfo{}, err
	}

	info := app.UserInfo{
		Login:     response.Login,
		FirstName: response.FirstName,
		LastName:  response.LastName,
		Address:   response.Address,
	}

	return info, nil
}

type userProfileResponse struct {
	Login     string `json:"login"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Address   string `json:"address"`
}
