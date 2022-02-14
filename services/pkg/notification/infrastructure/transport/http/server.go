package http

import (
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/jwtauth"
	"arch-homework/pkg/notification/app"
	"io/ioutil"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

const PathPrefix = "/api/v1/"

const (
	notificationsEndpoint = PathPrefix + "notifications"
)

const (
	errorCodeUnknown = 0
)

const authTokenHeader = "X-Auth-Token"

var errForbidden = errors.New("access forbidden")

func NewServer(notificationRepo app.NotificationRepository, tokenParser jwtauth.TokenParser, logger *logrus.Logger) *Server {
	return &Server{
		notificationRepo: notificationRepo,
		tokenParser:      tokenParser,
		logger:           logger,
	}
}

type Server struct {
	notificationRepo app.NotificationRepository
	tokenParser      jwtauth.TokenParser
	logger           *logrus.Logger
}

func (s *Server) MakeHandler() http.Handler {
	router := mux.NewRouter()

	router.Methods(http.MethodGet).Path(notificationsEndpoint).Handler(s.makeHandlerFunc(s.getNotificationsHandler))

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

func (s *Server) getNotificationsHandler(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}

	notifications, err := s.notificationRepo.FindAllByUserID(app.UserID(tokenData.UserID()))
	if err != nil {
		return err
	}
	notificationInfos := make([]notificationInfo, 0, len(notifications))
	for _, notification := range notifications {
		info, err := toNotificationInfo(notification)
		if err != nil {
			return err
		}
		notificationInfos = append(notificationInfos, info)
	}
	writeResponse(w, notificationInfos)
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
	case errForbidden:
		w.WriteHeader(http.StatusForbidden)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	js, _ := json.Marshal(info)
	_, _ = w.Write(js)
}

func toNotificationInfo(notification app.Notification) (notificationInfo, error) {
	info := notificationInfo{
		Type:         string(notification.Type),
		Message:      notification.Message,
		CreationDate: notification.CreationDate.Format(time.RFC3339),
	}
	if notification.LotID != nil {
		info.LotID = string(*notification.LotID)
	}

	return info, nil
}

type errorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type notificationInfo struct {
	Type         string `json:"type"`
	LotID        string `json:"lotId,omitempty"`
	Message      string `json:"message"`
	CreationDate string `json:"creationDate"`
}
