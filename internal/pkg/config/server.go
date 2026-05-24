package config

import (
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/ilyakaznacheev/cleanenv"
)

type ConfigServer struct {
	App      AppServer
	Database DatabaseServer
	Secrets  SecretsServer
	Auth     AuthConfig
}

type AuthConfig struct {
	Username string `env:"AUTH_USERNAME" env-default:"admin"`
	Password string `env:"AUTH_PASSWORD" env-default:"12345678"`
}

type AppServer struct {
	AppName  string `env:"APP_NAME" env-default:"envgate"`
	GRPCPort int32  `env:"GRPC_PORT" env-default:"50051"`
	RESTPort int32  `env:"REST_PORT" env-default:"3000"`
}

type DatabaseServer struct {
	DBPath string `env:"DB_PATH" env-default:"envgate.db"`
}

type SecretsServer struct {
	MasterKey string `env:"MASTER_KEY" env-required:"true"`
}

func (c *ConfigServer) GetAESKey() []byte {
	hash := sha256.Sum256([]byte(c.Secrets.MasterKey))
	return hash[:]
}

func MustLoadConfigServer() *ConfigServer {
	var cfg ConfigServer
	if err := cleanenv.ReadConfig(".env", &cfg); err != nil {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			panic(fmt.Errorf("failed to load server config: %w", err))
		}
	}

	log.Printf("Loaded config: %+v\n", cfg)
	return &cfg
}
