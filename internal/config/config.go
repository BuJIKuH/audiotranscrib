package config

import (
	"log"
	"os"
)

type Config struct {
	TelegramToken string `env:"TELEGRAM_TOKEN,required"`
	DatabaseDNS   string `env:"DATABASE_DNS,required"`
}

func InitConfig() *Config {

	cfg := &Config{
		TelegramToken: os.Getenv("TELEGRAM_TOKEN"),
		DatabaseDNS:   os.Getenv("DATABASE_DNS"),
	}

	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN is required")
	}

	return cfg
}
