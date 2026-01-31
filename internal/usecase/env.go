package usecase

import (
	"github.com/spf13/viper"
)

func getEnv(key, fallback string) string {
	if value := viper.GetString(key); value != "" {
		return value
	}
	return fallback
}
