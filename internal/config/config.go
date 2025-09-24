package config

import (
	"os"
)

type Config struct {
	Rate  RateConfig
	Bybit BybitConfig
}

type RateConfig struct {
	APIURL string
	APIKey string
}

type BybitConfig struct {
	APIURL    string
	APIKey    string
	APISecret string
}

func Load() *Config {
	return &Config{
		Rate: RateConfig{
			APIURL: getEnv("RATE_API_URL", "https://api.rate.com"),
			APIKey: getEnv("RATE_API_KEY", ""),
		},
		Bybit: BybitConfig{
			APIURL:    getEnv("BYBIT_API_URL", "https://api.bybit.com"),
			APIKey:    getEnv("BYBIT_API_KEY", ""),
			APISecret: getEnv("BYBIT_API_SECRET", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}