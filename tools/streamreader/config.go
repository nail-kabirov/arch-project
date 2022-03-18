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
	if c.RMQHost == "" || c.RMQPort == "" || c.RMQUser == "" || c.RMQPassword == "" {
		return c, errors.New("rabbit mq env params not set")
	}
	return c, nil
}

type config struct {
	StreamName string `envconfig:"stream_name" default:"integration_event"`

	RMQHost     string `envconfig:"rmq_host" default:"localhost"`
	RMQPort     string `envconfig:"rmq_port" default:"5552"`
	RMQUser     string `envconfig:"rmq_user" default:"rmq_user"`
	RMQPassword string `envconfig:"rmq_password" default:"rmq_pwd"`
}
