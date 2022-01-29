package main

import (
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
)

func parseEnv() (*config, error) {
	c := new(config)
	if err := envconfig.Process("", c); err != nil {
		return nil, errors.Wrap(err, "failed to parse env")
	}
	if c.DBHost == "" || c.DBPort == "" || c.DBName == "" || c.DBUser == "" || c.DBPassword == "" {
		return c, errors.New("db env params not set")
	}
	return c, nil
}

type config struct {
	ServicePort string `envconfig:"service_port" default:"8000"`
	JWTSecret   string `envconfig:"jwt_secret" default:"secret"`

	DBHost     string `envconfig:"db_host" default:"localhost"`
	DBPort     string `envconfig:"db_port" default:"5433"`
	DBName     string `envconfig:"db_name" default:"auth_db"`
	DBUser     string `envconfig:"db_user" default:"auth_user"`
	DBPassword string `envconfig:"db_password" default:"auth-pwd"`

	RedisHost     string `envconfig:"redis_host" default:"localhost"`
	RedisPort     string `envconfig:"redis_port" default:"6379"`
	RedisPassword string `envconfig:"redis_password" default:"redis-pwd"`
}
