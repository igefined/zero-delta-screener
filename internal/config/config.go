package config

import (
	"os"
	"strings"
)

type Config struct {
	Gate            GateConfig
	ByBit           ByBitConfig
	SupportedTokens []string
}

type GateConfig struct {
	WsURL     string
	APIKey    string
	APISecret string
}

type ByBitConfig struct {
	WsURL     string
	APIKey    string
	APISecret string
}

func Load() *Config {
	return &Config{
		Gate: GateConfig{
			WsURL:     getEnv("GATE_WS_URL", "wss://api.gateio.ws/ws/v4/"),
			APIKey:    getEnv("GATE_API_KEY", ""),
			APISecret: getEnv("GATE_API_SECRET", ""),
		},
		ByBit: ByBitConfig{
			WsURL:     getEnv("BYBIT_WS_URL", "wss://stream.bybit.com/v5/public/spot"),
			APIKey:    getEnv("BYBIT_API_KEY", ""),
			APISecret: getEnv("BYBIT_API_SECRET", ""),
		},
		SupportedTokens: getSupportedTokens(),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getSupportedTokens() []string {
	tokensEnv := getEnv("SUPPORTED_TOKENS", "")
	if tokensEnv != "" {
		return strings.Split(tokensEnv, ",")
	}

	return []string{"WLFI_USDT"}
}
