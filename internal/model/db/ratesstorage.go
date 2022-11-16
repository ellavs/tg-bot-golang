package db

// Работа с хранилищем курсов валют.

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ellavs/tg-bot-golang/internal/helpers/dbutils"
	types "github.com/ellavs/tg-bot-golang/internal/model/bottypes"
)

// ExchangeRatesStorage Тип для хранилища курсов валют.
type ExchangeRatesStorage struct {
	db                  *sqlx.DB
	usedCurrenciesNames []string
}

// RateDB Тип, принимающий структуру таблицы курсов валют.
type RateDB struct {
	Name string  `db:"currency"`
	Rate float64 `db:"rate"`
}

// NewExchangeRatesStorage Инициализация хранилища курсов валют.
// db - *sqlx.DB - ссылка на подключение к БД.
// usedCurrenciesNames - []string - массив используемых в приложении валют.
func NewExchangeRatesStorage(db *sqlx.DB, usedCurrenciesNames []string) *ExchangeRatesStorage {
	return &ExchangeRatesStorage{
		db:                  db,
		usedCurrenciesNames: usedCurrenciesNames,
	}
}

// InsertExchangeRatesToDate Добавление курсов валют в базу данных
func (storage *ExchangeRatesStorage) InsertExchangeRatesToDate(ctx context.Context, rates types.ExchangeRate, period time.Time) error {
	// Преобразование map в слайсы по используемым валютам для удобства вставки в БД.
	ratesNames := make([]string, len(storage.usedCurrenciesNames)-1)
	ratesValues := make([]float64, len(storage.usedCurrenciesNames)-1)
	for ind, cName := range storage.usedCurrenciesNames {
		if rate, ok := rates[cName]; ok {
			ratesNames[ind] = cName
			ratesValues[ind] = rate
		}
	}

	// Запрос на добавление данных.
	const sqlString = `
		INSERT INTO exchangerates (currency, rate, period)
			SELECT *, $1 FROM unnest($2::text[], $3::float[])
			 ON CONFLICT (currency, period) DO NOTHING`

	// Выполнение запроса на добавление данных.
	_, err := dbutils.Exec(ctx, storage.db, sqlString, period, ratesNames, ratesValues)
	if err != nil {
		return err
	}

	return nil
}

// GetLastExchangeRates Получение последних курсов валют из базы данных.
func (storage *ExchangeRatesStorage) GetLastExchangeRates(ctx context.Context) (types.ExchangeRate, error) {
	// Отбор последних курсов заданных валют.
	const sqlString = `
		SELECT rt.currency, rt.rate
		FROM exchangerates AS rt
         INNER JOIN (SELECT currency,
                            MAX(period) AS maxperiod
                     FROM exchangerates
                     WHERE currency = ANY($1)
                     GROUP BY currency)
                    AS rtMaxPeriod
                    ON rt.period = rtMaxPeriod.maxperiod
                        AND rt.currency = rtMaxPeriod.currency;`

	var rates []RateDB
	// Выполнение запроса на выборку данных (запись в переменную rates).
	if err := dbutils.Select(ctx, storage.db, &rates, sqlString, storage.usedCurrenciesNames); err != nil {
		return nil, err
	}

	exchangeRates := types.ExchangeRate{}
	for _, rate := range rates {
		exchangeRates[rate.Name] = rate.Rate
	}
	return exchangeRates, nil
}
