package http

import (
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/infrastructure/metrics"
	"arch-homework/pkg/common/jwtauth"
	"arch-homework/pkg/delivery/app"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

const PathPrefix = "/api/v1/"

const (
	lotSentEndpoint          = PathPrefix + "lot/sent"
	lotReceivedEndpoint      = PathPrefix + "lot/received"
	specificDeliveryEndpoint = PathPrefix + "lot/{id}/delivery"
)

const (
	errorCodeUnknown          = 0
	errorCodeInvalidRequestID = 1
	errorCodeAlreadyProcessed = 2
	errorCodeUserNotFound     = 3
	errorCodeInvalidLotStatus = 4
)

const authTokenHeader = "X-Auth-Token"
const requestIDHeader = "X-Request-ID"

var errForbidden = errors.New("access forbidden")
var errInvalidRequestID = errors.New("empty or invalid request id")

func NewEndpointLabelCollector() metrics.EndpointLabelCollector {
	return endpointLabelCollector{}
}

type endpointLabelCollector struct {
}

func (e endpointLabelCollector) EndpointLabelForURI(uri string) string {
	if strings.HasPrefix(uri, PathPrefix) {
		r, _ := regexp.Compile("^" + PathPrefix + "lot/[a-f0-9-]+/delivery")
		if r.MatchString(uri) {
			return specificDeliveryEndpoint
		}
	}
	return uri
}

func NewServer(
	userService *app.DeliveryService,
	tokenParser jwtauth.TokenParser,
	logger *logrus.Logger,
) *Server {
	return &Server{
		deliveryService: userService,
		tokenParser:     tokenParser,
		logger:          logger,
	}
}

type Server struct {
	deliveryService *app.DeliveryService
	tokenParser     jwtauth.TokenParser
	logger          *logrus.Logger
}

func (s *Server) MakeHandler() http.Handler {
	router := mux.NewRouter()

	router.Methods(http.MethodPost).Path(lotSentEndpoint).Handler(s.makeHandlerFunc(s.lotSentHandler))
	router.Methods(http.MethodPost).Path(lotReceivedEndpoint).Handler(s.makeHandlerFunc(s.lotReceivedHandler))
	router.Methods(http.MethodGet).Path(specificDeliveryEndpoint).Handler(s.makeHandlerFunc(s.getLotDeliveryHandler))

	return router
}

func (s *Server) MakeInternalHandler() http.Handler {
	router := mux.NewRouter()
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

func (s *Server) lotSentHandler(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	var lotSentData lotSentData
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &lotSentData); err != nil {
		return errors.WithStack(err)
	}

	requestID, err := s.getRequestIDHeader(r)
	if err != nil {
		return err
	}

	err = s.deliveryService.SetLotSent(requestID, app.UserID(tokenData.UserID()), app.LotID(lotSentData.LotID), app.TrackingID(lotSentData.TrackingID))
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, http.StatusText(http.StatusOK))
	return nil
}

func (s *Server) lotReceivedHandler(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	var lotReceivedData lotReceivedData
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &lotReceivedData); err != nil {
		return errors.WithStack(err)
	}

	requestID, err := s.getRequestIDHeader(r)
	if err != nil {
		return err
	}

	err = s.deliveryService.SetLotReceived(requestID, app.UserID(tokenData.UserID()), app.LotID(lotReceivedData.LotID))
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, http.StatusText(http.StatusOK))
	return nil
}

func (s *Server) getLotDeliveryHandler(w http.ResponseWriter, r *http.Request) error {
	_, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	lotID, err := getLotIDFromRequest(r)
	if err != nil {
		return err
	}

	profile, err := s.deliveryService.LotDeliveryInfo(lotID)
	if err != nil {
		return err
	}
	writeResponse(w, toDeliveryInfo(*profile))
	return nil
}

func (s *Server) extractAuthorizationData(r *http.Request) (jwtauth.TokenData, error) {
	token := r.Header.Get(authTokenHeader)
	if token == "" {
		return nil, errors.WithStack(errForbidden)
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

func getLotIDFromRequest(r *http.Request) (app.LotID, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return "", errors.WithStack(errors.New("id param required"))
	}
	if err := uuid.ValidateUUID(id); err != nil {
		return "", errors.WithStack(err)
	}
	return app.LotID(id), nil
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
	case app.ErrInvalidLotStatus:
		info.Code = errorCodeInvalidLotStatus
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrAlreadyProcessed:
		info.Code = errorCodeAlreadyProcessed
		w.WriteHeader(http.StatusConflict)
	case app.ErrLotNotFound:
		info.Code = errorCodeUserNotFound
		w.WriteHeader(http.StatusNotFound)
	case errForbidden, app.ErrStatusChangeForbidden:
		w.WriteHeader(http.StatusForbidden)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	js, _ := json.Marshal(info)
	_, _ = w.Write(js)
}

func toDeliveryInfo(info app.DeliveryInfo) deliveryInfo {
	trackingID := ""
	if info.TrackingID != nil {
		trackingID = string(*info.TrackingID)
	}
	return deliveryInfo{
		LotID:      string(info.LotID),
		Status:     string(info.LotStatus),
		TrackingID: trackingID,
		Sender: senderInfo{
			Login:     info.SenderLogin,
			FirstName: info.SenderFirstName,
			LastName:  info.SenderLastName,
		},
		Receiver: receiverInfo{
			Login:     info.ReceiverLogin,
			FirstName: info.ReceiverFirstName,
			LastName:  info.ReceiverLastName,
			Address:   string(info.ReceiverAddress),
		},
	}
}

type errorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type lotSentData struct {
	LotID      string `json:"id"`
	TrackingID string `json:"trackingId"`
}

type lotReceivedData struct {
	LotID string `json:"id"`
}

type senderInfo struct {
	Login     string `json:"login"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type receiverInfo struct {
	Login     string `json:"login"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Address   string `json:"address"`
}

type deliveryInfo struct {
	LotID      string       `json:"id"`
	Status     string       `json:"status"`
	Sender     senderInfo   `json:"sender"`
	Receiver   receiverInfo `json:"receiver"`
	TrackingID string       `json:"trackingId,omitempty"`
}
