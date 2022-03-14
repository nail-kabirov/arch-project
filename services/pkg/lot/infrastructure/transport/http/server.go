package http

import (
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/infrastructure/metrics"
	"arch-homework/pkg/common/jwtauth"
	"io"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"arch-homework/pkg/lot/app"
)

const PathPrefix = "/api/v1/"
const PathPrefixInternal = "/internal/api/v1/"

const (
	createLotEndpoint           = PathPrefix + "lot"
	createBidEndpoint           = PathPrefix + "lot/{id}/bid"
	lotsEndpoint                = PathPrefix + "lots"
	myLotsEndpoint              = PathPrefix + "lots/my"
	specificLotEndpoint         = PathPrefix + "lot/{id}"
	internalSpecificLotEndpoint = PathPrefixInternal + "lot/{id}"
)

const (
	errorCodeUnknown              = 0
	errorCodeInvalidRequestID     = 1
	errorCodeAlreadyProcessed     = 2
	errorCodeOrderNotFound        = 3
	errorCodePaymentFailed        = 4
	errorCodeInvalidEndTime       = 5
	errorCodeInvalidBuyItNowPrice = 6
	errorCodeLotClosed            = 7
	errorCodeInvalidBidAmount     = 8
	errorCodeInvalidAmount        = 9
	errorBidOnOwnLot              = 10
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
		if r, _ := regexp.Compile("^" + PathPrefix + "lot/[a-f0-9-]+/bid$"); r.MatchString(uri) {
			return createBidEndpoint
		}
		if r, _ := regexp.Compile("^" + PathPrefix + "lot/[a-f0-9-]+$"); r.MatchString(uri) {
			return specificLotEndpoint
		}
		if r, _ := regexp.Compile("^" + PathPrefix + "lots[?]"); r.MatchString(uri) {
			return lotsEndpoint
		}
	} else if strings.HasPrefix(uri, PathPrefixInternal) {
		r, _ := regexp.Compile("^" + PathPrefixInternal + "lot/[a-f0-9-]+$")
		if r.MatchString(uri) {
			return internalSpecificLotEndpoint
		}
	}
	return uri
}

func NewServer(lotService app.LotService, lotQueryService app.LotQueryService, tokenParser jwtauth.TokenParser, logger *logrus.Logger) *Server {
	return &Server{
		lotService:      lotService,
		lotQueryService: lotQueryService,
		tokenParser:     tokenParser,
		logger:          logger,
	}
}

type Server struct {
	lotService      app.LotService
	lotQueryService app.LotQueryService
	tokenParser     jwtauth.TokenParser
	logger          *logrus.Logger
}

func (s *Server) MakeHandler() http.Handler {
	router := mux.NewRouter()

	router.Methods(http.MethodPost).Path(createLotEndpoint).Handler(s.makeHandlerFunc(s.createLotHandler))
	router.Methods(http.MethodPost).Path(createBidEndpoint).Handler(s.makeHandlerFunc(s.createBidHandler))
	router.Methods(http.MethodGet).Path(specificLotEndpoint).Handler(s.makeHandlerFunc(s.getLotHandler))
	router.Methods(http.MethodGet).Path(lotsEndpoint).Handler(s.makeHandlerFunc(s.findLotsHandler))
	router.Methods(http.MethodGet).Path(myLotsEndpoint).Handler(s.makeHandlerFunc(s.myLotsHandler))

	return router
}

func (s *Server) MakeInternalHandler() http.Handler {
	router := mux.NewRouter()
	router.Methods(http.MethodGet).Path(internalSpecificLotEndpoint).Handler(s.makeHandlerFunc(s.getLotInternalHandler))
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

func (s *Server) getLotHandler(w http.ResponseWriter, r *http.Request) error {
	_, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}
	lotID, err := getIDFromRequest(r)
	if err != nil {
		return err
	}

	lot, err := s.lotQueryService.Get(lotID)
	if err != nil {
		return err
	}
	writeResponse(w, toLotInfo(*lot))
	return nil
}

func (s *Server) getLotInternalHandler(w http.ResponseWriter, r *http.Request) error {
	lotID, err := getIDFromRequest(r)
	if err != nil {
		return err
	}

	lot, err := s.lotQueryService.Get(lotID)
	if err != nil {
		return err
	}
	writeResponse(w, toLotInfo(*lot))
	return nil
}

func (s *Server) findLotsHandler(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	var createdAfter *time.Time
	var searchString *string
	withParticipationOnly := false
	wonOnly := false

	query := r.URL.Query()
	if after := query.Get("createdAfter"); after != "" {
		afterTime, err := time.Parse(time.RFC3339, after)
		if err != nil {
			return errors.WithStack(err)
		}
		createdAfter = &afterTime
	}
	if search := query.Get("search"); search != "" {
		searchString = &search
	}
	if participation := query.Get("participation"); participation == "1" {
		withParticipationOnly = true
	}
	if win := query.Get("win"); win == "1" {
		wonOnly = true
	}

	lots, err := s.lotQueryService.FindAvailable(app.UserID(tokenData.UserID()), createdAfter, searchString, withParticipationOnly, wonOnly)
	if err != nil {
		return err
	}
	lotInfos := make([]lotInfo, 0, len(lots))
	for _, lot := range lots {
		lotInfos = append(lotInfos, toLotInfo(lot))
	}
	writeResponse(w, lotInfos)
	return nil
}

func (s *Server) myLotsHandler(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	lots, err := s.lotQueryService.FindByOwnerID(app.UserID(tokenData.UserID()))
	if err != nil {
		return err
	}
	lotInfos := make([]lotExInfo, 0, len(lots))
	for _, lot := range lots {
		bids := make([]bidInfo, 0, len(lot.Bids))
		for _, bid := range lot.Bids {
			bids = append(bids, bidInfo{
				UserID:       string(bid.UserID),
				UserLogin:    bid.UserLogin,
				Amount:       bid.Amount.Value(),
				CreationDate: bid.CreationTime.Format(time.RFC3339),
			})
		}
		info := lotExInfo{
			ID:           string(lot.ID),
			Description:  lot.Description,
			EndTime:      lot.EndTime.Format(time.RFC3339),
			StartPrice:   lot.StartPrice.Value(),
			Status:       string(lot.Status),
			CreationDate: lot.CreationTime.Format(time.RFC3339),
			Bids:         bids,
		}
		if lot.BuyItNowPrice != nil {
			info.BuyItNowPrice = (*lot.BuyItNowPrice).Value()
		}
		lotInfos = append(lotInfos, info)
	}
	writeResponse(w, lotInfos)
	return nil
}

func (s *Server) createLotHandler(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	requestID, err := s.getRequestIDHeader(r)
	if err != nil {
		return err
	}

	var info createLotInfo
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &info); err != nil {
		return err
	}

	endTime, err := time.Parse(time.RFC3339, info.EndTime)
	if err != nil {
		return errors.WithStack(err)
	}
	var buyItNowPrice *float64
	if info.BuyItNowPrice != 0 {
		buyItNowPrice = &info.BuyItNowPrice
	}

	lotID, err := s.lotService.CreateLot(
		requestID,
		app.UserID(tokenData.UserID()),
		info.Description,
		info.StartPrice,
		endTime,
		buyItNowPrice,
	)
	if err != nil {
		return err
	}
	response := createLotResponse{ID: string(lotID)}
	writeResponse(w, response)
	return nil
}

func (s *Server) createBidHandler(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	requestID, err := s.getRequestIDHeader(r)
	if err != nil {
		return err
	}

	lotID, err := getIDFromRequest(r)
	if err != nil {
		return err
	}

	var info createBidInfo
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &info); err != nil {
		return err
	}

	err = s.lotService.CreateBid(requestID, app.UserID(tokenData.UserID()), lotID, info.Amount)
	if err != nil {
		return err
	}

	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, http.StatusText(http.StatusOK))
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

func getIDFromRequest(r *http.Request) (app.LotID, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return "", errors.New("id param required")
	}
	if err := uuid.ValidateUUID(id); err != nil {
		return "", err
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
	case app.ErrAlreadyProcessed:
		info.Code = errorCodeAlreadyProcessed
		w.WriteHeader(http.StatusConflict)
	case app.ErrLotNotFound:
		info.Code = errorCodeOrderNotFound
		w.WriteHeader(http.StatusNotFound)
	case app.ErrPaymentFailed:
		info.Code = errorCodePaymentFailed
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrInvalidEndTime:
		info.Code = errorCodeInvalidEndTime
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrInvalidBuyItNowPrice:
		info.Code = errorCodeInvalidBuyItNowPrice
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrLotClosed:
		info.Code = errorCodeLotClosed
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrInvalidBidAmount:
		info.Code = errorCodeInvalidBidAmount
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrNegativeAmount, app.ErrNotRoundedAmount:
		info.Code = errorCodeInvalidAmount
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrBidOnOwnLot:
		info.Code = errorBidOnOwnLot
		w.WriteHeader(http.StatusBadRequest)
	case errForbidden:
		w.WriteHeader(http.StatusForbidden)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	js, _ := json.Marshal(info)
	_, _ = w.Write(js)
}

func toLotInfo(lot app.LotQueryData) lotInfo {
	info := lotInfo{
		ID:            string(lot.ID),
		Description:   lot.Description,
		EndTime:       lot.EndTime.Format(time.RFC3339),
		StartPrice:    lot.StartPrice.Value(),
		Status:        string(lot.Status),
		OwnerID:       string(lot.OwnerID),
		OwnerLogin:    lot.OwnerLogin,
		CreationDate:  lot.CreationTime.Format(time.RFC3339),
		LastBidAmount: 0,
		LastBidderID:  "",
	}
	if lot.BuyItNowPrice != nil {
		info.BuyItNowPrice = (*lot.BuyItNowPrice).Value()
	}
	if lot.LastBidAmount != nil {
		info.LastBidAmount = (*lot.LastBidAmount).Value()
	}
	if lot.LastBidderID != nil {
		info.LastBidderID = string(*lot.LastBidderID)
	}
	return info
}

type errorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type lotInfo struct {
	ID            string  `json:"id"`
	Description   string  `json:"description"`
	EndTime       string  `json:"endTime"`
	StartPrice    float64 `json:"startPrice"`
	BuyItNowPrice float64 `json:"buyItNowPrice,omitempty"`
	Status        string  `json:"status"`
	OwnerID       string  `json:"ownerId"`
	OwnerLogin    string  `json:"ownerLogin"`
	CreationDate  string  `json:"creationDate"`
	LastBidAmount float64 `json:"lastBidAmount,omitempty"`
	LastBidderID  string  `json:"lastBidderId,omitempty"`
}

type bidInfo struct {
	UserID       string  `json:"userId"`
	UserLogin    string  `json:"userLogin"`
	Amount       float64 `json:"amount"`
	CreationDate string  `json:"creationDate"`
}

type lotExInfo struct {
	ID            string    `json:"id"`
	Description   string    `json:"description"`
	EndTime       string    `json:"endTime"`
	StartPrice    float64   `json:"startPrice"`
	BuyItNowPrice float64   `json:"buyItNowPrice,omitempty"`
	Status        string    `json:"status"`
	CreationDate  string    `json:"creationDate"`
	Bids          []bidInfo `json:"bids"`
}

type createLotInfo struct {
	Description   string  `json:"description"`
	EndTime       string  `json:"endTime"`
	StartPrice    float64 `json:"startPrice"`
	BuyItNowPrice float64 `json:"buyItNowPrice,omitempty"`
}

type createBidInfo struct {
	Amount float64 `json:"amount"`
}

type createLotResponse struct {
	ID string `json:"id"`
}
