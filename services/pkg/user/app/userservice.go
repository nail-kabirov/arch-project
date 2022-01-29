package app

import (
	"arch-homework/pkg/common/app/storedevent"
	"net/mail"

	"github.com/pkg/errors"
)

var ErrAlreadyProcessed = errors.New("request with this id already processed")

func NewUserService(dbDependency DBDependency, eventSender storedevent.Sender, authSvcClient AuthServiceClient) *UserService {
	return &UserService{
		readRepo:      dbDependency.UserProfileRepositoryRead(),
		trUnitFactory: dbDependency,
		eventSender:   eventSender,
		authSvcClient: authSvcClient,
	}
}

type UserService struct {
	readRepo      UserProfileRepositoryRead
	trUnitFactory TransactionalUnitFactory
	eventSender   storedevent.Sender
	authSvcClient AuthServiceClient
}

func (s *UserService) Add(login, password string, firstName, lastName string, email Email, address Address) (UserID, error) {
	if err := s.checkEmail(email, nil); err != nil {
		return "", errors.WithStack(err)
	}

	id, err := s.authSvcClient.RegisterUser(login, password)
	if err != nil {
		return "", err
	}

	userProfile := UserProfile{
		UserID:    id,
		Login:     login,
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Address:   address,
	}

	err = s.executeInTransaction(func(provider RepositoryProvider) error {
		err2 := provider.UserProfileRepository().Store(&userProfile)
		if err2 != nil {
			return err2
		}

		event := NewUserRegisteredEvent(id, login)
		err2 = provider.EventStore().Add(event)
		if err2 != nil {
			return err2
		}
		s.eventSender.EventStored(event.UID)

		return nil
	})
	if err != nil {
		_ = s.authSvcClient.RemoveUser(id)
		return "", err
	}

	s.eventSender.SendStoredEvents()
	return id, err
}

func (s *UserService) Update(requestID RequestID, id UserID, firstName, lastName *string, email *Email, address *Address) error {
	if email != nil {
		if err := s.checkEmail(*email, &id); err != nil {
			return errors.WithStack(err)
		}
	}

	return s.executeInTransaction(func(provider RepositoryProvider) error {
		eventRepo := provider.ProcessedRequestRepository()
		alreadyProcessed, err := eventRepo.SetRequestProcessed(requestID)
		if err != nil {
			return err
		}
		if alreadyProcessed {
			return ErrAlreadyProcessed
		}

		profileRepo := provider.UserProfileRepository()
		user, err := profileRepo.FindByID(id)
		if err != nil {
			return errors.WithStack(err)
		}
		if firstName != nil {
			user.FirstName = *firstName
		}
		if lastName != nil {
			user.LastName = *lastName
		}
		if email != nil {
			user.Email = *email
		}
		if address != nil {
			user.Address = *address
		}
		return profileRepo.Store(user)
	})
}

func (s *UserService) GetUserProfile(id UserID) (*UserProfile, error) {
	user, err := s.readRepo.FindByID(id)
	return user, errors.WithStack(err)
}

func (s *UserService) checkEmail(email Email, userID *UserID) error {
	if _, err := mail.ParseAddress(string(email)); err != nil {
		return errors.Wrap(ErrInvalidEmail, err.Error())
	}

	if user, err := s.readRepo.FindByEmail(email); err != ErrUserNotFound || user != nil {
		if user != nil && (userID == nil || *userID != user.UserID) {
			return ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

func (s *UserService) executeInTransaction(f func(RepositoryProvider) error) (err error) {
	var trUnit TransactionalUnit
	trUnit, err = s.trUnitFactory.NewTransactionalUnit()
	if err != nil {
		return err
	}
	defer func() {
		err = trUnit.Complete(err)
	}()
	err = f(trUnit)
	return err
}
