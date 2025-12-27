package help

import (
	"log/slog"
	"os"
)

func Logger() *slog.Logger {
	// Level can come from config/env; Info is a good production default.
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		// AddSource: false, // keep off in prod unless you need it
	}

	h := slog.NewJSONHandler(os.Stdout, opts)

	log := slog.New(h).With(
		slog.String("service", "ashCache"),
		slog.String("env", "test"),
	)

	// Optional: make it the default logger used by slog.Info/Debug/etc.
	slog.SetDefault(log)

	return log
}
