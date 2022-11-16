// Package dbutils Хелпер-обёртка для выполнения запросов на базе sqlx и для функций подключения к БД (pgx).
package dbutils

// Хелпер-обёртка для функций подключения к БД (pgx)

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/ellavs/tg-bot-golang/internal/logger"
)

// pgxLogger Логгер для pgx, реализующий интерфейс Logger пакета pgx.
type pgxLogger struct{}

// Log Функция реализации интерфейса Logger пакета pgx.
func (pl *pgxLogger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]any) {
	var buffer bytes.Buffer
	buffer.WriteString(msg)
	for k, v := range data {
		buffer.WriteString(fmt.Sprintf(" %s=%+v", k, v))
	}
	switch level {
	case pgx.LogLevelTrace, pgx.LogLevelNone, pgx.LogLevelDebug:
		logger.Debug(buffer.String())
	case pgx.LogLevelInfo:
		logger.Info(buffer.String())
	case pgx.LogLevelWarn:
		logger.Warn(buffer.String())
	case pgx.LogLevelError:
		logger.Error(buffer.String())
	default:
		logger.Debug(buffer.String())
	}
}

// NewDBConnect Инициализация подключения к базе данных по заданным параметрам.
func NewDBConnect(connString string) (*sqlx.DB, error) {
	connConfig, err := pgx.ParseConfig(connString)
	if err != nil {
		logger.Error("Ошибка парсинга строки подключения", "err", err)
		return nil, err
	}
	connConfig.RuntimeParams["application_name"] = "tg-bot"
	connConfig.Logger = &pgxLogger{}
	connConfig.LogLevel = pgx.LogLevelDebug
	connStr := stdlib.RegisterConnConfig(connConfig)
	dbh, err := sqlx.Connect("pgx", connStr)
	if err != nil {
		logger.Error("Ошибка соединения с БД", "err", err)
		return nil, fmt.Errorf("Ошибка: prepare db connection: %w", err)
	}
	return dbh, nil
}
