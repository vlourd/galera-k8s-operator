package config

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Name        string `required:"false" envconfig:"NAME"`
	Version     string `required:"false" envconfig:"VERSION" default:"betav1"`
	StateAPIURL string `required:"false" envconfig:"STATE_API"`
}

func ConfigFromEnv() (*Config, error) {
	conf := &Config{}

	err := envconfig.Process("", conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
