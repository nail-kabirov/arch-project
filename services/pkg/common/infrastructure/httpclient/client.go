package httpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

const (
	jsonContentType = "application/json"
	requestIDHeader = "X-Request-ID"
)

type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s: %s", http.StatusText(e.StatusCode), e.Body)
}

func NewClient(client http.Client, host string) Client {
	return &httpClient{client: client, host: host}
}

type Client interface {
	MakeJSONRequest(request, response interface{}, method, reqURL string, requestID *string) error
}

type httpClient struct {
	client http.Client
	host   string
}

func (client *httpClient) MakeJSONRequest(request, response interface{}, method, reqURL string, requestID *string) error {
	var bodyReader io.Reader
	if request != nil {
		body, err := json.Marshal(request)
		if err != nil {
			return errors.WithStack(err)
		}
		bodyReader = bytes.NewReader(body)
	}
	resBody, err := client.makeRequest(bodyReader, jsonContentType, method, reqURL, requestID)
	if err != nil {
		return err
	}

	if response != nil && len(resBody) > 0 {
		return errors.WithStack(json.Unmarshal(resBody, response))
	}
	return nil
}

func (client *httpClient) makeRequest(bodyReader io.Reader, contentType, method, reqURL string, requestID *string) ([]byte, error) {
	req, err := http.NewRequest(method, client.host+reqURL, bodyReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if bodyReader != nil {
		req.Header.Set("Content-Type", contentType)
	}
	if requestID != nil {
		req.Header.Set(requestIDHeader, *requestID)
	}

	res, err := client.client.Do(req)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		return nil, &HTTPError{StatusCode: res.StatusCode, Body: string(body)}
	}
	if res.Body == nil {
		return nil, nil
	}

	resBody, err := io.ReadAll(res.Body)
	return resBody, errors.WithStack(err)
}
