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
	if c.RMQHost == "" || c.RMQPort == "" || c.RMQUser == "" || c.RMQPassword == "" {
		return c, errors.New("rabbit mq env params not set")
	}
	return c, nil
}

type config struct {
	ServicePort string `envconfig:"service_port" default:"8000"`
	JWTSecret   string `envconfig:"jwt_secret" default:"secret"`

	LotServiceHost  string `envconfig:"lot_host" default:"http://lot-app:8000"`
	UserServiceHost string `envconfig:"user_host" default:"http://user-app:8000"`

	DBHost     string `envconfig:"db_host" default:"localhost"`
	DBPort     string `envconfig:"db_port" default:"5433"`
	DBName     string `envconfig:"db_name" default:"delivery_db"`
	DBUser     string `envconfig:"db_user" default:"delivery_user"`
	DBPassword string `envconfig:"db_password" default:"delivery-pwd"`

	RMQHost     string `envconfig:"rmq_host" default:"localhost"`
	RMQPort     string `envconfig:"rmq_port" default:"5552"`
	RMQUser     string `envconfig:"rmq_user" default:"rmq_user"`
	RMQPassword string `envconfig:"rmq_password" default:"rmq_pwd"`
}
