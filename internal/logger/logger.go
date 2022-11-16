package logger

import (
	"log"

	"go.uber.org/zap"
)

// Глобальная переменная логгера.
var logger *zap.Logger

// init Инициализация логгера для использования его во всем приложении.
// init будет выполнен один раз, независимо от количества импортов в разных местах приложения.
func init() {
	// Инициализация для режима разработки.
	localLogger, err := zap.NewDevelopment()
	// Вариант инициализации для продакшена.
	//localLogger, err := zap.NewProduction()

	if err != nil {
		log.Fatal("Ошибка инициализации логгера zap", err)
	}

	logger = localLogger

}

// Fatal - запись в лог, уровень Fatal.
func Fatal(msg string, keysAndValues ...interface{}) {
	sugar := logger.Sugar()
	sugar.Fatalw(msg, keysAndValues...)
}

// Error - запись в лог, уровень Error.
func Error(msg string, keysAndValues ...interface{}) {
	sugar := logger.Sugar()
	sugar.Errorw(msg, keysAndValues...)
}

// Warn - запись в лог, уровень Warn.
func Warn(msg string, keysAndValues ...interface{}) {
	sugar := logger.Sugar()
	sugar.Warnw(msg, keysAndValues...)
}

// Info - запись в лог, уровень Info.
func Info(msg string, keysAndValues ...interface{}) {
	sugar := logger.Sugar()
	sugar.Infow(msg, keysAndValues...)
}

// Debug - запись в лог, уровень Debug.
func Debug(msg string, keysAndValues ...interface{}) {
	sugar := logger.Sugar()
	sugar.Debugw(msg, keysAndValues...)
}

// DebugZap - запись в лог, уровень DebugZap.
func DebugZap(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}
