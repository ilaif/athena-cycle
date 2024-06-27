package logging

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/rs/zerolog"
)

func NewLogger() (logr.Logger, error) {
	zl := zerolog.New(zerolog.NewConsoleWriter())
	zl = zl.With().Caller().Timestamp().Logger()
	return zerologr.New(&zl), nil
}

func NewContext(ctx context.Context, log logr.Logger) context.Context {
	return logr.NewContext(ctx, log)
}

func MustFromContext(ctx context.Context) logr.Logger {
	log, err := logr.FromContext(ctx)
	if err != nil {
		panic(err)
	}
	return log
}
