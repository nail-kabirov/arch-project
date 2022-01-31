package http

import (
	"arch-homework/pkg/billing/app"
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/jwtauth"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const PathPrefix = "/api/v1/"
const PathPrefixInternal = "/internal/api/v1/"

const (
	accountEndpoint = PathPrefix + "account"
	paymentEndpoint = PathPrefixInternal + "payment"
)

const (
	errorCodeUnknown              = 0
	errorCodeInvalidRequestID     = 1
	errorCodeAlreadyProcessed     = 2
	errorNotEnoughFunds           = 3
	errorInvalidAmount            = 4
	errorLotPaymentAlreadyBlocked = 5
)

const authTokenHeader = "X-Auth-Token"
const requestIDHeader = "X-Request-ID"

var errForbidden = errors.New("access forbidden")
var errInvalidRequestID = errors.New("empty or invalid request id")

func NewServer(billingService app.BillingService, billingQueryService app.BillingQueryService, tokenParser jwtauth.TokenParser, logger *logrus.Logger) *Server {
	return &Server{
		billingService:      billingService,
		billingQueryService: billingQueryService,
		tokenParser:         tokenParser,
		logger:              logger,
	}
}

type Server struct {
	billingService      app.BillingService
	billingQueryService app.BillingQueryService
	tokenParser         jwtauth.TokenParser
	logger              *logrus.Logger
}

func (s *Server) MakeHandler() http.Handler {
	router := mux.NewRouter()
	router.Methods(http.MethodGet).Path(accountEndpoint).Handler(s.makeHandlerFunc(s.getAccountStatusEndpoint))
	router.Methods(http.MethodPost).Path(accountEndpoint).Handler(s.makeHandlerFunc(s.topUpAccountEndpoint))
	return router
}

func (s *Server) MakeInternalHandler() http.Handler {
	router := mux.NewRouter()
	router.Methods(http.MethodPost).Path(paymentEndpoint).Handler(s.makeHandlerFunc(s.processPaymentEndpoint))
	return router
}

func (s *Server) makeHandlerFunc(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		fields := logrus.Fields{
			"method": r.Method,
			"host":   r.Host,
			"path":   r.URL.Path,
		}
		if r.URL.RawQuery != "" {
			fields["query"] = r.URL.RawQuery
		}
		if r.PostForm != nil {
			fields["post"] = r.PostForm
		}

		if r.Body != nil {
			bytesBody, _ := ioutil.ReadAll(r.Body)
			_ = r.Body.Close()
			if len(bytesBody) > 0 {
				r.Body = ioutil.NopCloser(bytes.NewBuffer(bytesBody))
				fields["body"] = string(bytesBody)
			}
		}
		headersBytes, _ := json.Marshal(r.Header)
		fields["headers"] = string(headersBytes)

		err := fn(w, r)

		if err != nil {
			writeErrorResponse(w, err)

			fields["err"] = err
			s.logger.WithFields(fields).Error()
		} else {
			s.logger.WithFields(fields).Info("call")
		}
	}
}

func (s *Server) getAccountStatusEndpoint(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	status, err := s.billingQueryService.AccountBalance(app.UserID(tokenData.UserID()))
	if err != nil {
		return err
	}
	writeResponse(w, accountStatusResponse{
		Amount:        status.Amount.Value(),
		BlockedAmount: status.BlockedAmount.Value(),
	})
	return nil
}

func (s *Server) topUpAccountEndpoint(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	requestID, err := s.getRequestIDHeader(r)
	if err != nil {
		return err
	}

	var info topUpAccountInfo
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &info); err != nil {
		return err
	}
	amount, err := app.AmountFromFloat(info.Amount)
	if err != nil {
		return err
	}

	if err = s.billingService.TopUpAccount(requestID, app.UserID(tokenData.UserID()), amount); err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func (s *Server) processPaymentEndpoint(w http.ResponseWriter, r *http.Request) error {
	requestID, err := s.getRequestIDHeader(r)
	if err != nil {
		return err
	}

	var info paymentInfo
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &info); err != nil {
		return err
	}
	if err = uuid.ValidateUUID(info.UserID); err != nil {
		return err
	}
	if err = uuid.ValidateUUID(info.LotID); err != nil {
		return err
	}
	amount, err := app.AmountFromFloat(info.Amount)
	if err != nil {
		return err
	}

	if err = s.billingService.ProcessLotPayment(requestID, app.UserID(info.UserID), app.LotID(info.LotID), amount); err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

func (s *Server) extractAuthorizationData(r *http.Request) (jwtauth.TokenData, error) {
	token := r.Header.Get(authTokenHeader)
	if token == "" {
		return nil, errForbidden
	}
	tokenData, err := s.tokenParser.ParseToken(token)
	if err != nil {
		return nil, errors.Wrap(errForbidden, err.Error())
	}
	if err = uuid.ValidateUUID(tokenData.UserID()); err != nil {
		return nil, errors.WithStack(err)
	}
	return tokenData, nil
}

func (s *Server) getRequestIDHeader(r *http.Request) (app.RequestID, error) {
	requestID := r.Header.Get(requestIDHeader)
	err := uuid.ValidateUUID(requestID)
	if err != nil {
		return "", errors.Wrap(errInvalidRequestID, err.Error())
	}
	return app.RequestID(requestID), nil
}

func writeResponse(w http.ResponseWriter, response interface{}) {
	js, err := json.Marshal(response)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(js)
}

func writeErrorResponse(w http.ResponseWriter, err error) {
	info := errorInfo{Code: errorCodeUnknown, Message: err.Error()}
	switch errors.Cause(err) {
	case errInvalidRequestID:
		info.Code = errorCodeInvalidRequestID
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrAlreadyProcessed:
		info.Code = errorCodeAlreadyProcessed
		w.WriteHeader(http.StatusConflict)
	case app.ErrBlockPayment:
		info.Code = errorNotEnoughFunds
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrLotPaymentAlreadyBlocked:
		info.Code = errorLotPaymentAlreadyBlocked
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrNegativeAmount, app.ErrNotRoundedAmount:
		info.Code = errorInvalidAmount
		w.WriteHeader(http.StatusBadRequest)
	case errForbidden:
		w.WriteHeader(http.StatusForbidden)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	js, _ := json.Marshal(info)
	_, _ = w.Write(js)
}

type errorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type accountStatusResponse struct {
	Amount        float64 `json:"amount"`
	BlockedAmount float64 `json:"blockedAmount"`
}

type topUpAccountInfo struct {
	Amount float64 `json:"amount"`
}

type paymentInfo struct {
	UserID string  `json:"userId"`
	LotID  string  `json:"lotId"`
	Amount float64 `json:"amount"`
}
