package postgres

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

func NewClient(db *sqlx.DB) DBClient {
	return &client{DB: db}
}

type Client interface {
	Select(dest interface{}, query string, args ...interface{}) error
	Get(dest interface{}, query string, args ...interface{}) error
	Exec(query string, args ...interface{}) (sql.Result, error)

	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
}

type Transaction interface {
	Client
	Commit() error
	Rollback() error
}

type TransactionalClient interface {
	Client
	BeginTransaction() (Transaction, error)
}

type DBClient interface {
	TransactionalClient
	Close() error
	Ping() error
}

type client struct {
	*sqlx.DB
}

func (c *client) BeginTransaction() (Transaction, error) {
	return c.Beginx()
}
