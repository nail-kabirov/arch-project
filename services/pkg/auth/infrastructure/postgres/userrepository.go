package postgres

import (
	"arch-homework/pkg/auth/app"
	"arch-homework/pkg/common/infrastructure/postgres"

	"database/sql"

	"github.com/pkg/errors"
)

func NewUserRepository(client postgres.Client) app.UserRepository {
	return &userRepository{client: client}
}

type userRepository struct {
	client postgres.Client
}

func (repo *userRepository) Store(user *app.User) error {
	const query = `
			INSERT INTO auth_user (id, login, password)
			VALUES (:id, :login, :password)
			ON CONFLICT (id) DO UPDATE SET
				login = excluded.login,
				password = excluded.password
		`

	userx := sqlxUser{
		ID:       string(user.UserID),
		Login:    string(user.Login),
		Password: string(user.Password),
	}

	_, err := repo.client.NamedExec(query, &userx)
	return errors.WithStack(err)
}

func (repo *userRepository) Remove(id app.UserID) error {
	const query = `DELETE FROM auth_user WHERE id = $1`
	_, err := repo.client.Exec(query, string(id))
	return errors.WithStack(err)
}

func (repo *userRepository) FindByID(id app.UserID) (*app.User, error) {
	const query = `SELECT id, login, password FROM auth_user WHERE id = $1`

	var user sqlxUser
	err := repo.client.Get(&user, query, string(id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrUserNotFound
		}
		return nil, errors.WithStack(err)
	}
	res := sqlxUserToUser(&user)
	return &res, nil
}

func (repo *userRepository) FindByLogin(login app.Login) (*app.User, error) {
	const query = `SELECT id, login, password FROM auth_user WHERE login = $1`

	var user sqlxUser
	err := repo.client.Get(&user, query, string(login))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, app.ErrUserNotFound
		}
		return nil, errors.WithStack(err)
	}
	res := sqlxUserToUser(&user)
	return &res, nil
}

func sqlxUserToUser(user *sqlxUser) app.User {
	return app.User{
		UserID:   app.UserID(user.ID),
		Login:    app.Login(user.Login),
		Password: app.Password(user.Password),
	}
}

type sqlxUser struct {
	ID       string `db:"id"`
	Login    string `db:"login"`
	Password string `db:"password"`
}
