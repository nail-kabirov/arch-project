package user

import (
	"arch-homework/pkg/common/infrastructure/httpclient"
	"arch-homework/pkg/lot/app"

	"fmt"
	"net/http"
)

const userProfileURLTpl = "/internal/api/v1/user/%s/profile"

func NewClient(client http.Client, serviceHost string) app.UserClient {
	return &userClient{
		httpClient:   httpclient.NewClient(client, serviceHost),
		userLoginMap: make(map[app.UserID]string),
	}
}

type userClient struct {
	httpClient   httpclient.Client
	userLoginMap map[app.UserID]string
}

func (c *userClient) GetUserLogin(userID app.UserID) (string, error) {
	if login, ok := c.userLoginMap[userID]; ok {
		return login, nil
	}

	requestURL := fmt.Sprintf(userProfileURLTpl, string(userID))
	var response userProfileResponse
	err := c.httpClient.MakeJSONRequest(nil, &response, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}
	c.userLoginMap[userID] = response.Login
	return response.Login, nil
}

type userProfileResponse struct {
	Login string `json:"login"`
}
