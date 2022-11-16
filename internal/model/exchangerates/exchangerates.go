package exchangerates

// Пакет для работы с курсами валют

import (
	"context"
	"github.com/ellavs/tg-bot-golang/internal/logger"
	"sync"
	"time"

	"github.com/pkg/errors"
	types "github.com/ellavs/tg-bot-golang/internal/model/bottypes"
)

type CbrClient interface {
	// Загрузка курсов валют из открытого источника.
	LoadExchangeRates() (types.ExchangeRate, time.Time, error)
}

// Интерфейс для работы с хранилищем курсов валют.
type RatesDataStorage interface {
	// Запись курсов валют в базу данных на определенную дату.
	InsertExchangeRatesToDate(ctx context.Context, rates types.ExchangeRate, period time.Time) error
	// Получение последних курсов валют из базы данных.
	GetLastExchangeRates(ctx context.Context) (types.ExchangeRate, error)
}

// Структура для хранения текущей информации о валютах.
type ExchangeRates struct {
	sync.RWMutex
	Ctx              context.Context
	IsLoaded         bool               // Флаг, что хотя бы одна загрузка была произведена.
	LoadDateTime     time.Time          // Время последней загрузки.
	Rates            types.ExchangeRate // Последние курсы валют (кэш-данных из БД).
	CurrenciesName   []string           // Массив используемых валют.
	MainCurrency     string             // Основная валюта для хранения данных.
	CbrClient        CbrClient          // Источник загружаемых курсов валют.
	RatesDataStorage RatesDataStorage   // Хранилище курсов валют.
}

// Инициализация экземпляра класса информации о курсах валют.
func New(ctx context.Context, cbrClient CbrClient, currenciesName []string, mainCurrency string, storage RatesDataStorage) *ExchangeRates {
	exchangeRatesStorage := ExchangeRates{
		Ctx:              ctx,
		Rates:            types.ExchangeRate{},
		CbrClient:        cbrClient,
		CurrenciesName:   currenciesName,
		MainCurrency:     mainCurrency,
		RatesDataStorage: storage,
	}
	// курс базовой валюты устанавливаем в единицу.
	exchangeRatesStorage.Rates[mainCurrency] = 1
	return &exchangeRatesStorage
}

// Получение курса указанной валюты.
func (currenciesStorage *ExchangeRates) GetExchangeRate(currencyName string) (float64, error) {
	currenciesStorage.RLock()
	// Попытка получить курс из локального кэша.
	if currenciesStorage.IsLoaded {
		if rate, ok := currenciesStorage.Rates[currencyName]; ok {
			currenciesStorage.RUnlock()
			return rate, nil
		}
	}
	currenciesStorage.RUnlock()

	// Валюты нет в локальном кэше, попытка принудительно синхронно загрузить курсы валют из хранилища (из базы данных).
	if err := currenciesStorage.LoadExchangeRatesFromStorage(); err != nil {
		logger.Error("Ошибка получения курсов валют из БД", "err", err)
		return 0, err
	}

	// Валюты нет в хранилище (в БД), попытка принудительно синхронно загрузить курсы валют из внешнего источника.
	if err := currenciesStorage.UpdateExchangeRates(); err != nil {
		logger.Error("Ошибка загрузки курсов валют", "err", err)
		return 0, err
	}

	// Попытка получить курс из кэша после принудительного синхронного обновления курсов в кэше.
	currenciesStorage.RLock()
	defer currenciesStorage.RUnlock()
	if rate, ok := currenciesStorage.Rates[currencyName]; ok {
		return rate, nil
	}
	return 0, errors.New("Курс валюты получить не удалось.")
}

// Получение названия основной валюты.
func (currenciesStorage *ExchangeRates) GetMainCurrency() string {
	return currenciesStorage.MainCurrency
}

// Получение списка используемых валют.
func (currenciesStorage *ExchangeRates) GetCurrenciesList() []string {
	return currenciesStorage.CurrenciesName
}

// Конвертация суммы в указанную валюту из базовой валюты.
func (currenciesStorage *ExchangeRates) ConvertSumFromBaseToCurrency(currencyName string, sum int64) (int64, error) {
	return currenciesStorage.convertSum(currencyName, sum, false)
}

// Конвертация суммы из указанной валюты в базовую валюту.
func (currenciesStorage *ExchangeRates) ConvertSumFromCurrencyToBase(currencyName string, sum int64) (int64, error) {
	return currenciesStorage.convertSum(currencyName, sum, true)
}

// Конвертация суммы из/в базовую валюту в/из указанной.
// Флаг isFrom == true означает конвертацию из указанной валюты в базовую.
func (currenciesStorage *ExchangeRates) convertSum(currencyName string, sum int64, isFrom bool) (int64, error) {
	if sum == 0 {
		return 0, nil
	}
	if currenciesStorage.MainCurrency == currencyName {
		return sum, nil
	}
	currentRate, err := currenciesStorage.GetExchangeRate(currencyName)
	if err != nil {
		logger.Error("Ошибка получения курса валюты", "err", err)
		return 0, err
	}
	if currentRate == 0 {
		logger.Error("Ошибка указания курса валюты (0)")
		return 0, errors.New("Курс валюты некорректный.")
	}
	var res int64
	if isFrom {
		// Конвертация из указанной валюты в базовую.
		res = int64(float64(sum) / currentRate)
	} else {
		// Конвертация из базовой валюты в указанную.
		res = int64(float64(sum) * currentRate)
	}
	return res, nil
}

// Загрузка курсов валют из внешнего источника.
func (curStorage *ExchangeRates) UpdateExchangeRates() error {
	// Загрузка курсов из заданного источника.
	curExchangeRates, period, err := curStorage.CbrClient.LoadExchangeRates()
	if err != nil {
		logger.Error("Ошибка загрузки курсов валют", "err", err)
		return err
	}
	// Обновление курсов в локальном кэше, если загрузка прошла успешно.
	curStorage.Lock()
	curStorage.IsLoaded = true
	curStorage.LoadDateTime = period
	for _, cName := range curStorage.CurrenciesName {
		if rate, ok := curExchangeRates[cName]; ok {
			curStorage.Rates[cName] = rate
		}
	}
	curStorage.Unlock()
	// Обновление курсов в хранилище (БД).
	if err := curStorage.RatesDataStorage.InsertExchangeRatesToDate(curStorage.Ctx, curExchangeRates, curStorage.LoadDateTime); err != nil {
		logger.Error("Ошибка обновления курсов валют", "err", err)
		return err
	}
	return nil
}

// Получение курсов валют из хранилища (БД) в локальный кэш.
func (curStorage *ExchangeRates) LoadExchangeRatesFromStorage() error {
	// Загрузка последних курсов из хранилища.
	curExchangeRates, err := curStorage.RatesDataStorage.GetLastExchangeRates(curStorage.Ctx)
	if err != nil {
		logger.Error("Ошибка загрузки последних курсов валют", "err", err)
		return err
	}
	// Обновление курсов в локальном кэше, если загрузка прошла успешно.
	curStorage.Lock()
	curStorage.IsLoaded = true
	curStorage.LoadDateTime = time.Now()
	for _, cName := range curStorage.CurrenciesName {
		if rate, ok := curExchangeRates[cName]; ok {
			curStorage.Rates[cName] = rate
		}
	}
	curStorage.Unlock()
	return nil
}
