package postgres

import (
	"arch-homework/pkg/common/infrastructure/postgres"
	"arch-homework/pkg/user/app"

	"database/sql"

	"github.com/pkg/errors"
)

func NewUserProfileRepository(client postgres.Client) app.UserProfileRepository {
	return &userProfileRepository{client: client}
}

type userProfileRepository struct {
	client postgres.Client
}

func (repo *userProfileRepository) Store(profile *app.UserProfile) error {
	const query = `
			INSERT INTO user_profile (id, login, first_name, last_name, email, address)
			VALUES (:id, :login, :first_name, :last_name, :email, :address)
			ON CONFLICT (id) DO UPDATE SET
				first_name = excluded.first_name,
				last_name = excluded.last_name,
				email = excluded.email,
				address = excluded.address;
		`

	profilex := sqlxUserProfile{
		ID:        string(profile.UserID),
		Login:     profile.Login,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Email:     string(profile.Email),
		Address:   string(profile.Address),
	}

	_, err := repo.client.NamedExec(query, &profilex)
	return errors.WithStack(err)
}

func (repo *userProfileRepository) Remove(id app.UserID) error {
	const query = `DELETE FROM user_profile WHERE id = $1`
	_, err := repo.client.Exec(query, string(id))
	return errors.WithStack(err)
}

func (repo *userProfileRepository) FindByID(id app.UserID) (*app.UserProfile, error) {
	const query = `SELECT id, login, first_name, last_name, email, address FROM user_profile WHERE id = $1`

	var profile sqlxUserProfile
	err := repo.client.Get(&profile, query, string(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrUserNotFound
		}
		return nil, errors.WithStack(err)
	}
	res := sqlxProfileToProfile(profile)
	return &res, nil
}

func (repo *userProfileRepository) FindByEmail(email app.Email) (*app.UserProfile, error) {
	const query = `SELECT id, login, first_name, last_name, email, address FROM user_profile WHERE email = $1`

	var profile sqlxUserProfile
	err := repo.client.Get(&profile, query, string(email))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrUserNotFound
		}
		return nil, errors.WithStack(err)
	}
	res := sqlxProfileToProfile(profile)
	return &res, nil
}

func sqlxProfileToProfile(profile sqlxUserProfile) app.UserProfile {
	return app.UserProfile{
		UserID:    app.UserID(profile.ID),
		Login:     profile.Login,
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		Email:     app.Email(profile.Email),
		Address:   app.Address(profile.Address),
	}
}

type sqlxUserProfile struct {
	ID        string `db:"id"`
	Login     string `db:"login"`
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
	Address   string `db:"address"`
}
