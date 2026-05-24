package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type ConfigAgent struct {
	Server ServerTarget
	Auth   AuthAgent
}

type ServerTarget struct {
	Address  string `env:"ENVGATE_SERVER_ADDR" env-default:"localhost:50051"`
	Insecure bool   `env:"ENVGATE_SERVER_INSECURE" env-default:"true"`
}

type AuthAgent struct {
	Token string `env:"ENVGATE_TOKEN" env-required:"true"`
}

func MustLoadConfigAgent() *ConfigAgent {
	var cfg ConfigAgent
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic(fmt.Errorf("failed to load agent config: %w", err))
	}
	return &cfg
}
