package config

import (
	"github.com/caarlos0/env/v6"
)

type Config struct {
	Address           string `env:"RUN_ADDRESS,required"`
	DBURL             string `env:"DATABASE_URI,required"`
	ProcessingAddress string `env:"ACCRUAL_SYSTEM_ADDRESS,required"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)

	return &cfg, err
}
