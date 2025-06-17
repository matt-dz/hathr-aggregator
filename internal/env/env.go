// Package for environmental dependencies

package env

import (
	"play-aggregator/internal/httpclient"
	"play-aggregator/internal/logging"

	"log/slog"
)

const Key = "hathr-env"

// Holds the dependencies for the environment
type Env struct {
	Logger     *slog.Logger
	HttpClient *httpclient.Client
}

// Constructs an Env object with the provided parameters
func New(logger *slog.Logger, httpclient *httpclient.Client) *Env {
	if logger == nil {
		logger = slog.New(logging.NullLogger())
	}

	return &Env{
		Logger:     logger,
		HttpClient: httpclient,
	}
}

// Constructs a null instance
func Null() *Env {
	return &Env{
		Logger: slog.New(logging.NullLogger()),
	}
}
