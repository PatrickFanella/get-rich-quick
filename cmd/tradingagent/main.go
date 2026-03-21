package main

import (
	"log/slog"
	"os"

	"github.com/PatrickFanella/get-rich-quick/internal/config"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}

	logger := config.SetDefaultLogger(env, level)
	logger.Info("starting trading agent",
		slog.String("env", env),
		slog.String("log_level", level),
	)
}
