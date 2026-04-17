package config

import (
	"log"
	"os"
)

type Config struct {
	TelegramToken   string `env:"TELEGRAM_TOKEN,required"`
	DatabaseDNS     string `env:"DATABASE_DNS,required"`
	SaluteSpeechKey string `env:"SALUTE_SPEECH_KEY,required"`
	GigaChatKey     string `env:"GIGA_CHAT_KEY,required"`
}

func InitConfig() *Config {

	cfg := &Config{
		TelegramToken:   os.Getenv("TELEGRAM_TOKEN"),
		DatabaseDNS:     os.Getenv("DATABASE_DNS"),
		SaluteSpeechKey: os.Getenv("SALUTE_SPEECH_KEY"),
		GigaChatKey:     os.Getenv("GIGA_CHAT_KEY"),
	}

	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_TOKEN is required")
	}

	if cfg.DatabaseDNS == "" {
		log.Fatal("DATABASE_DNS is required")
	}
	if cfg.SaluteSpeechKey == "" {
		log.Fatal("SALUTE_SPEECH_KEY is required")
	}
	if cfg.GigaChatKey == "" {
		log.Fatal("GIGA_CHAT_KEY is required")
	}

	return cfg
}
