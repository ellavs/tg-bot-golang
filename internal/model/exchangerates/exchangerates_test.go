package exchangerates

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mocks "github.com/ellavs/tg-bot-golang/internal/mocks/exchangerates"
	types "github.com/ellavs/tg-bot-golang/internal/model/bottypes"
)

func Test_UpdateExchangeRates_ShouldWithoutError(t *testing.T) {
	// Тестирование процедуры загрузки курсов валют.
	period := time.Now()
	ctx := context.Background()

	// Имитируем ответ сервиса курсов валют.
	ctrl := gomock.NewController(t)
	cbrClient := mocks.NewMockCbrClient(ctrl)
	// Ожидаем вызова загрузки курса валют.
	cbrClient.EXPECT().LoadExchangeRates().Return(types.ExchangeRate{"USD": 0.0006, "CNY": 0.012}, period, nil)

	// Имитируем наличие базы данных.
	ctrlStorage := gomock.NewController(t)
	ratesDataStorage := mocks.NewMockRatesDataStorage(ctrlStorage)
	// Имитация сохранения курсов в БД.
	ratesDataStorage.EXPECT().InsertExchangeRatesToDate(ctx, types.ExchangeRate{"USD": 0.0006, "CNY": 0.012}, period).Return(nil)

	// Запускаем тест
	exchangeRates := New(ctx, cbrClient, []string{"USD", "CNY", "EUR", "RUB"}, "RUB", ratesDataStorage)
	err := exchangeRates.UpdateExchangeRates()

	assert.NoError(t, err)
	assert.Equal(t,
		&ExchangeRates{
			IsLoaded:         true,
			Rates:            types.ExchangeRate{"USD": 0.0006, "CNY": 0.012, "RUB": 1},
			CurrenciesName:   []string{"USD", "CNY", "EUR", "RUB"},
			MainCurrency:     "RUB",
			CbrClient:        cbrClient,
			LoadDateTime:     period,
			Ctx:              ctx,
			RatesDataStorage: ratesDataStorage,
		},
		exchangeRates,
	)
}

// Тестирование функции конвертации валют.
func Test_ConvertSumFromBaseToCurrency_RUB_ShouldWithoutError(t *testing.T) {
	exchangeRates := New(context.Background(), nil, []string{"USD", "CNY", "EUR", "RUB"}, "RUB", nil)
	exchangeRates.Rates = types.ExchangeRate{"USD": 0.0006, "CNY": 0.012, "RUB": 1}

	var testSum int64 = 10000 // 100 рублей
	res, err := exchangeRates.ConvertSumFromBaseToCurrency("RUB", testSum)

	assert.NoError(t, err)
	assert.Equal(t, testSum, res) // 100 рублей
}

// Тестирование функции конвертации валют.
func Test_ConvertSumFromBaseToCurrency_USD_ShouldWithoutError(t *testing.T) {
	exchangeRates := New(context.Background(), nil, []string{"USD", "CNY", "EUR", "RUB"}, "RUB", nil)
	exchangeRates.Rates = types.ExchangeRate{"USD": 0.016, "CNY": 0.1, "RUB": 1}
	exchangeRates.IsLoaded = true

	var testSum int64 = 10000 // 100 рублей
	res, err := exchangeRates.ConvertSumFromBaseToCurrency("USD", testSum)

	assert.NoError(t, err)
	assert.Equal(t, int64(160), res) // 1 доллар 16 центов
}
