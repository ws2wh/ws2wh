package logger

import (
	"log/slog"
	"os"

	"github.com/ws2wh/ws2wh/server"
)

func InitLogger(config *server.Config) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     config.LogLevel,
		AddSource: true,
	})
	slog.SetDefault(slog.New(handler))
}
