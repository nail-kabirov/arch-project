package http

import (
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/infrastructure/metrics"
	"arch-homework/pkg/common/jwtauth"
	"arch-homework/pkg/user/app"

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
const PathPrefixInternal = "/internal/api/v1/"

const (
	registerUserEndpoint = PathPrefix + "register"
	userProfileEndpoint  = PathPrefix + "user/profile"

	internalSpecificUserProfileEndpoint = PathPrefixInternal + "user/{id}/profile"
)

const (
	errorCodeUnknown          = 0
	errorCodeInvalidRequestID = 1
	errorCodeAlreadyProcessed = 2
	errorCodeUserNotFound     = 3
	errorEmailAlreadyExists   = 4
	errorInvalidEmail         = 5
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
	if strings.HasPrefix(uri, PathPrefixInternal) {
		r, _ := regexp.Compile("^" + PathPrefixInternal + "user/[a-f0-9-]+/profile$")
		if r.MatchString(uri) {
			return internalSpecificUserProfileEndpoint
		}
	}
	return uri
}

func NewServer(
	userService *app.UserService,
	tokenParser jwtauth.TokenParser,
	logger *logrus.Logger,
) *Server {
	return &Server{
		userService: userService,
		tokenParser: tokenParser,
		logger:      logger,
	}
}

type Server struct {
	userService *app.UserService
	tokenParser jwtauth.TokenParser
	logger      *logrus.Logger
}

func (s *Server) MakeHandler() http.Handler {
	router := mux.NewRouter()

	router.Methods(http.MethodPost).Path(registerUserEndpoint).Handler(s.makeHandlerFunc(s.registerUserHandler))
	router.Methods(http.MethodGet).Path(userProfileEndpoint).Handler(s.makeHandlerFunc(s.getUserProfileHandler))
	router.Methods(http.MethodPut).Path(userProfileEndpoint).Handler(s.makeHandlerFunc(s.updateUserProfileHandler))

	return router
}

func (s *Server) MakeInternalHandler() http.Handler {
	router := mux.NewRouter()
	router.Methods(http.MethodGet).Path(internalSpecificUserProfileEndpoint).Handler(s.makeHandlerFunc(s.getUserProfileInternalHandler))
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

func (s *Server) registerUserHandler(w http.ResponseWriter, r *http.Request) error {
	var registerData registerUserData
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &registerData); err != nil {
		return errors.WithStack(err)
	}

	userID, err := s.userService.Add(
		registerData.Login,
		registerData.Password,
		registerData.FirstName,
		registerData.LastName,
		app.Email(registerData.Email),
		app.Address(registerData.Address),
	)
	if err != nil {
		return err
	}
	writeResponse(w, createdUserInfo{UserID: string(userID)})
	return nil
}

func (s *Server) getUserProfileHandler(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}
	id := app.UserID(tokenData.UserID())

	profile, err := s.userService.GetUserProfile(id)
	if err != nil {
		return err
	}
	writeResponse(w, toUserProfileInfo(*profile))
	return nil
}

func (s *Server) getUserProfileInternalHandler(w http.ResponseWriter, r *http.Request) error {
	id, err := getUserIDFromRequest(r)
	if err != nil {
		return err
	}
	profile, err := s.userService.GetUserProfile(id)
	if err != nil {
		return err
	}
	writeResponse(w, toUserProfileInfo(*profile))
	return nil
}

func (s *Server) updateUserProfileHandler(w http.ResponseWriter, r *http.Request) error {
	tokenData, err := s.extractAuthorizationData(r)
	if err != nil {
		return err
	}
	userID := app.UserID(tokenData.UserID())

	requestID, err := s.getRequestIDHeader(r)
	if err != nil {
		return errors.WithStack(err)
	}

	var info userInfoUpdate
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &info); err != nil {
		return errors.WithStack(err)
	}
	var email *app.Email
	var address *app.Address

	if info.Email != nil {
		emailValue := app.Email(*info.Email)
		email = &emailValue
	}
	if info.Address != nil {
		addressValue := app.Address(*info.Address)
		address = &addressValue
	}

	err = s.userService.Update(requestID, userID, info.FirstName, info.LastName, email, address)
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

func getUserIDFromRequest(r *http.Request) (app.UserID, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return "", errors.WithStack(errors.New("id param required"))
	}
	if err := uuid.ValidateUUID(id); err != nil {
		return "", errors.WithStack(err)
	}
	return app.UserID(id), nil
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
	case app.ErrInvalidEmail:
		info.Code = errorInvalidEmail
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrEmailAlreadyExists:
		info.Code = errorEmailAlreadyExists
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrAlreadyProcessed:
		info.Code = errorCodeAlreadyProcessed
		w.WriteHeader(http.StatusConflict)
	case app.ErrUserNotFound:
		info.Code = errorCodeUserNotFound
		w.WriteHeader(http.StatusNotFound)
	case errForbidden:
		w.WriteHeader(http.StatusForbidden)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	js, _ := json.Marshal(info)
	_, _ = w.Write(js)
}

func toUserProfileInfo(profile app.UserProfile) userInfo {
	return userInfo{
		UserID:    string(profile.UserID),
		Login:     profile.Login,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Email:     string(profile.Email),
		Address:   string(profile.Address),
	}
}

type errorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type registerUserData struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Address   string `json:"address"`
}

type userInfo struct {
	UserID    string `json:"id"`
	Login     string `json:"login"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Address   string `json:"address"`
}

type userInfoUpdate struct {
	UserID    *string `json:"id"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	Email     *string `json:"email"`
	Address   *string `json:"address"`
}

type createdUserInfo struct {
	UserID string `json:"id"`
}
