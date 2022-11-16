package cbr

// Пакет для загрузки курсов валют из источника:
// https://www.cbr-xml-daily.ru/latest.js

import (
	"context"
	"github.com/ellavs/tg-bot-golang/internal/logger"
	"time"

	types "github.com/ellavs/tg-bot-golang/internal/model/bottypes"
)

const currenciesURL = "https://www.cbr-xml-daily.ru/latest.js" // URL для загрузки курса валют.

type httpClient[T any] interface {
	GetJsonByURL(ctx context.Context, url string, jsonStruct *T) error
}

type CbrClient struct {
	ctx        context.Context
	httpClient httpClient[ExchangeRatesJson]
}

// Структура для загрузки курса валют из JSON.
type ExchangeRatesJson struct {
	Timestamp int64              `json:"timestamp"`
	Rates     types.ExchangeRate `json:"rates"`
}

func New(ctx context.Context, httpClient httpClient[ExchangeRatesJson]) *CbrClient {
	return &CbrClient{
		ctx:        ctx,
		httpClient: httpClient,
	}
}

// Загрузка курсов валют.
func (cbrClient *CbrClient) LoadExchangeRates() (types.ExchangeRate, time.Time, error) {
	// Переменная для загрузки данных из JSON.
	var curExchangeRates ExchangeRatesJson
	// Получение JSON данных по заданному URL и перенос их в указанную структуру.
	err := cbrClient.httpClient.GetJsonByURL(cbrClient.ctx, currenciesURL, &curExchangeRates)
	if err != nil {
		logger.Error("Ошибка получения данных курсов валют по URL", "err", err)
		return nil, time.Time{}, err
	}
	// Парсинг даты курса.
	period := time.Unix(curExchangeRates.Timestamp, 0)
	return curExchangeRates.Rates, period, nil
}
