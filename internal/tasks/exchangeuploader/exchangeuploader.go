package exchangeuploader

import (
	"context"
	"github.com/ellavs/tg-bot-golang/internal/logger"
	"time"
)

type ExchangeRates interface {
	UpdateExchangeRates() error
	LoadExchangeRatesFromStorage() error
}

// Процедура периодического обновления курсов валют из внешнего источника.
func ExchangeRatesUpdater(ctx context.Context, exchangeRatesStorage ExchangeRates, currenciesUpdatePeriod time.Duration) {
	// Создаем таймер на указанную периодичность.
	ticker := time.NewTicker(currenciesUpdatePeriod)
	// Запускаем горутину, обновляющую курсы валют по таймеру.
	go func() {
		for {
			select {
			case <-ctx.Done():
				// Завершение горутины.
				return
			case <-ticker.C:
				// Запуск процедуры загрузки.
				logger.Info("Загрузка курсов валют.")
				if err := exchangeRatesStorage.UpdateExchangeRates(); err != nil {
					logger.Error("Ошибка загрузки курсов валют:", "err", err)
				}
			}
		}
	}()
}

// Процедура периодической загрузки курсов валют из хранилища (БД) в локальный кэш.
func ExchangeRatesFromStorageLoader(ctx context.Context, exchangeRatesStorage ExchangeRates, currenciesUpdateCachePeriod time.Duration) {
	// Создаем таймер на указанную периодичность.
	ticker := time.NewTicker(currenciesUpdateCachePeriod)
	// Запускаем горутину, загружающую курсы валют по таймеру.
	go func() {
		for {
			select {
			case <-ctx.Done():
				// Завершение горутины.
				return
			case <-ticker.C:
				// Запуск процедуры загрузки.
				logger.Info("Загрузка курсов валют из БД в кэш.")
				if err := exchangeRatesStorage.LoadExchangeRatesFromStorage(); err != nil {
					logger.Error("Ошибка загрузки курсов валют из БД в кэш:", "err", err)
				}
			}
		}
	}()
}
