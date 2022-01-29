package http

import (
	"arch-homework/pkg/auth/app"
	"arch-homework/pkg/common/app/uuid"
	"arch-homework/pkg/common/infrastructure/metrics"
	"arch-homework/pkg/common/jwtauth"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
)

const PathPrefix = "/api/v1/"
const PathPrefixInternal = "/internal/api/v1/"

const (
	loginEndpoint  = PathPrefix + "login"
	logoutEndpoint = PathPrefix + "logout"

	internalRegisterUserEndpoint = PathPrefixInternal + "register"
	internalSpecificUserEndpoint = PathPrefixInternal + "user/{id}"
	internalAuthEndpoint         = PathPrefixInternal + "auth"
)

const (
	errorCodeUnknown               = 0
	errorCodeUserNotFound          = 1
	errorCodeUsernameAlreadyExists = 2
	errorCodeUsernameTooLong       = 3
	errorCodeInvalidPassword       = 4
)

const sessionCookieName = "session_id"
const authTokenHeader = "X-Auth-Token"

var errUnauthorized = errors.New("not authorized")

func NewEndpointLabelCollector() metrics.EndpointLabelCollector {
	return endpointLabelCollector{}
}

type endpointLabelCollector struct {
}

func (e endpointLabelCollector) EndpointLabelForURI(uri string) string {
	if strings.HasPrefix(uri, PathPrefixInternal) {
		r, _ := regexp.Compile("^" + PathPrefixInternal + "user/[a-f0-9-]+$")
		if r.MatchString(uri) {
			return internalSpecificUserEndpoint
		}
	}
	return uri
}

func NewServer(userService *app.UserService, sessionClient app.SessionClient, tokenGenerator jwtauth.TokenGenerator, logger *logrus.Logger) *Server {
	return &Server{
		userService:    userService,
		sessionClient:  sessionClient,
		tokenGenerator: tokenGenerator,
		logger:         logger,
	}
}

type Server struct {
	userService    *app.UserService
	sessionClient  app.SessionClient
	tokenGenerator jwtauth.TokenGenerator
	logger         *logrus.Logger
}

func (s *Server) MakeHandler() http.Handler {
	router := mux.NewRouter()
	router.Methods(http.MethodPost).Path(loginEndpoint).Handler(s.makeHandlerFunc(s.loginHandler))
	router.Methods(http.MethodPost).Path(logoutEndpoint).Handler(s.makeHandlerFunc(s.logoutHandler))
	return router
}

func (s *Server) MakeInternalHandler() http.Handler {
	router := mux.NewRouter()
	router.Path(internalAuthEndpoint).Handler(s.makeHandlerFunc(s.authHandler))
	router.Methods(http.MethodPost).Path(internalRegisterUserEndpoint).Handler(s.makeHandlerFunc(s.registerUserHandler))
	router.Methods(http.MethodDelete).Path(internalSpecificUserEndpoint).Handler(s.makeHandlerFunc(s.removeUserHandler))
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
	var info userAuthData
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &info); err != nil {
		return errors.WithStack(err)
	}

	userID, err := s.userService.Add(info.Login, info.Password)
	if err != nil {
		return err
	}
	writeResponse(w, createdUserInfo{UserID: string(userID)})
	return nil
}

func (s *Server) removeUserHandler(w http.ResponseWriter, r *http.Request) error {
	id, err := getUserIDFromRequest(r)
	if err != nil {
		return err
	}
	err = s.userService.Remove(id)
	if err != nil {
		return err
	}
	w.WriteHeader(http.StatusOK)
	return nil
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) error {
	var info userAuthData
	bytesBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.WithStack(err)
	}
	_ = r.Body.Close()
	if err = json.Unmarshal(bytesBody, &info); err != nil {
		return errors.WithStack(err)
	}

	user, err := s.userService.FindUserByLoginAndPassword(info.Login, info.Password)
	if err != nil {
		return err
	}
	session := app.Session{
		ID:     app.SessionID(uuid.GenerateNew()),
		UserID: user.UserID,
	}
	err = s.sessionClient.Store(session)
	if err != nil {
		return err
	}

	setSessionCookie(w, &session.ID)
	w.WriteHeader(http.StatusOK)
	return nil
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) error {
	if sessionID, err := getSessionIDFromRequest(r); err == nil {
		err = s.sessionClient.Remove(sessionID)
		if err != nil {
			return err
		}
	}

	setSessionCookie(w, nil)
	w.WriteHeader(http.StatusOK)
	return nil
}

func (s *Server) authHandler(w http.ResponseWriter, r *http.Request) error {
	sessionID, err := getSessionIDFromRequest(r)
	if err != nil {
		return errors.WithStack(errUnauthorized)
	}
	session, err := s.sessionClient.FindByID(sessionID)
	if err != nil {
		if errors.Cause(err) == app.ErrSessionNotFound {
			return errors.WithStack(errUnauthorized)
		}
		return err
	}
	user, err := s.userService.FindUserByID(session.UserID)
	if err != nil {
		return err
	}
	_ = s.sessionClient.UpdateSessionTTL(sessionID)

	token, err := s.tokenGenerator.GenerateToken(string(user.UserID), string(user.Login))
	if err != nil {
		return err
	}

	w.Header().Set(authTokenHeader, token)
	w.WriteHeader(http.StatusOK)
	return nil
}

func getUserIDFromRequest(r *http.Request) (app.UserID, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return "", errors.New("id param required")
	}
	if err := uuid.ValidateUUID(id); err != nil {
		return "", errors.WithStack(err)
	}
	return app.UserID(id), nil
}

func getSessionIDFromRequest(r *http.Request) (app.SessionID, error) {
	sessionID, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", err
	}
	if err := uuid.ValidateUUID(sessionID.Value); err != nil {
		return "", err
	}
	return app.SessionID(sessionID.Value), nil
}

func setSessionCookie(w http.ResponseWriter, sessionID *app.SessionID) {
	c := &http.Cookie{
		Name:     sessionCookieName,
		Path:     "/",
		HttpOnly: true,
	}
	if sessionID != nil {
		c.Value = string(*sessionID)
	} else {
		// delete cookie
		c.MaxAge = -1
	}

	http.SetCookie(w, c)
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
	case app.ErrUserNotFound:
		info.Code = errorCodeUserNotFound
		w.WriteHeader(http.StatusNotFound)
	case app.ErrLoginAlreadyExists:
		info.Code = errorCodeUsernameAlreadyExists
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrLoginTooLong:
		info.Code = errorCodeUsernameTooLong
		w.WriteHeader(http.StatusBadRequest)
	case app.ErrInvalidPassword:
		info.Code = errorCodeInvalidPassword
		w.WriteHeader(http.StatusBadRequest)
	case errUnauthorized:
		w.WriteHeader(http.StatusUnauthorized)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	js, _ := json.Marshal(info)
	_, _ = w.Write(js)
}

type userAuthData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type createdUserInfo struct {
	UserID string `json:"id"`
}

type errorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
