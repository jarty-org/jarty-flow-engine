package config

import (
	"os"
)

type Config struct {
}

var AppConfig Config

func LoadConfig() {
	config := Config{}
	AppConfig = config
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
