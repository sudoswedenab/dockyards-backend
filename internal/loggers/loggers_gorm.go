package loggers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/exp/slog"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

type gormSlogger struct {
	logger *slog.Logger
}

func (s *gormSlogger) LogMode(logLevel gormLogger.LogLevel) gormLogger.Interface {
	return s
}

func (s *gormSlogger) Info(ctx context.Context, format string, args ...interface{}) {
	s.logger.Info(fmt.Sprintf(format, args...))
}

func (s *gormSlogger) Warn(ctx context.Context, format string, args ...interface{}) {
	s.logger.Warn(fmt.Sprintf(format, args...))
}

func (s *gormSlogger) Error(ctx context.Context, format string, args ...interface{}) {
	s.logger.Error(fmt.Sprintf(format, args...))
}

func (s *gormSlogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	// ignore any record not found errors
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}

	if err != nil {
		sql, rows := fc()
		s.logger.Debug("gorm trace", "sql", sql, "rows", rows, "err", err)
	}
}

func NewGormSlogger(logger *slog.Logger) gormLogger.Interface {
	s := gormSlogger{
		logger: logger,
	}

	return &s
}
