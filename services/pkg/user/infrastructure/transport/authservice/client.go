package authservice

import (
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/infrastructure/httpclient"
	"arch-homework/pkg/user/app"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"net/http"
)

const registerUserURL = "/internal/api/v1/register"
const removeUserURLTemplate = "/internal/api/v1/user/%s"

func NewClient(client http.Client, serviceHost string) app.AuthServiceClient {
	return &authServiceClient{httpClient: httpclient.NewClient(client, serviceHost)}
}

type authServiceClient struct {
	httpClient httpclient.Client
}

func (c *authServiceClient) RegisterUser(login, password string) (app.UserID, error) {
	request := registerUserRequest{
		Login:    login,
		Password: password,
	}
	response := registerUserResponse{}
	err := c.httpClient.MakeJSONRequest(request, &response, http.MethodPost, registerUserURL, nil)
	if err != nil {
		if e, ok := errors.Cause(err).(*httpclient.HTTPError); ok {
			errInfo := errorInfo{}
			if json.Unmarshal([]byte(e.Body), &errInfo) == nil {
				return "", errors.WithStack(errors.New(errInfo.Message))
			}
		}
		return "", errors.WithStack(err)
	}

	if err = uuid.ValidateUUID(response.ID); err != nil {
		return "", errors.WithStack(err)
	}

	return app.UserID(response.ID), nil
}

func (c *authServiceClient) RemoveUser(userID app.UserID) error {
	url := fmt.Sprintf(removeUserURLTemplate, string(userID))
	err := c.httpClient.MakeJSONRequest(nil, nil, http.MethodDelete, url, nil)
	return errors.WithStack(err)
}

type registerUserRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type registerUserResponse struct {
	ID string `json:"id"`
}

type errorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
