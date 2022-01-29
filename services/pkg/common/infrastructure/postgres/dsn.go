package postgres

import "fmt"

type DSN struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
}

func (dsn *DSN) String() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dsn.User, dsn.Password, dsn.Host, dsn.Port, dsn.Database)
}
