package main

import (
	"context"
	"github.com/ellavs/tg-bot-golang/internal/cache"
	"github.com/ellavs/tg-bot-golang/internal/helpers/kafka"
	"github.com/ellavs/tg-bot-golang/internal/metrics"
	"github.com/ellavs/tg-bot-golang/internal/tasks/reportserver"
	"github.com/ellavs/tg-bot-golang/internal/tracing"
	"os/signal"
	"syscall"
	"time"

	"github.com/ellavs/tg-bot-golang/internal/clients/cbr"
	"github.com/ellavs/tg-bot-golang/internal/clients/tg"
	"github.com/ellavs/tg-bot-golang/internal/config"
	"github.com/ellavs/tg-bot-golang/internal/helpers/dbutils"
	"github.com/ellavs/tg-bot-golang/internal/helpers/net_http"
	"github.com/ellavs/tg-bot-golang/internal/logger"
	"github.com/ellavs/tg-bot-golang/internal/model/db"
	rates "github.com/ellavs/tg-bot-golang/internal/model/exchangerates"
	"github.com/ellavs/tg-bot-golang/internal/model/messages"
	uploader "github.com/ellavs/tg-bot-golang/internal/tasks/exchangeuploader"
)

// Параметры по умолчанию (могут быть изменены через config)
var (
	mainCurrency                = "RUB"                                // Основная валюта для хранения данных.
	currenciesName              = []string{"USD", "CNY", "EUR", "RUB"} // Массив используемых валют.
	currenciesUpdatePeriod      = 30 * time.Minute                     // Периодичность обновления курсов валют (раз в 30 минут).
	currenciesUpdateCachePeriod = 30 * time.Minute                     // Периодичность кэширования курсов валют из базы данных (раз в 30 минут).
	connectionStringDB          = ""                                   // Строка подключения к базе данных.
	kafkaTopic                  = "tgbot"                              // Наименование топика Kafka.
	brokersList                 = []string{"localhost:9092"}           // Список адресов брокеров сообщений (адрес Kafka).
)

func main() {

	logger.Info("Старт приложения")

	ctx := context.Background()

	config, err := config.New()
	if err != nil {
		logger.Fatal("Ошибка получения файла конфигурации:", "err", err)
	}

	// Изменение параметров по умолчанию из заданной конфигурации.
	setConfigSettings(config.GetConfig())

	// Оборачивание в Middleware функции обработки сообщения для метрик и трейсинга.
	tgProcessingFuncHandler := tg.HandlerFunc(tg.ProcessingMessages)
	tgProcessingFuncHandler = metrics.MetricsMiddleware(tgProcessingFuncHandler)
	tgProcessingFuncHandler = tracing.TracingMiddleware(tgProcessingFuncHandler)

	// Инициализация телеграм клиента.
	tgClient, err := tg.New(config, tgProcessingFuncHandler)
	if err != nil {
		logger.Fatal("Ошибка инициализации ТГ-клиента:", "err", err)
	}

	// Инициализация хранилищ (подключение к базе данных).
	dbconn, err := dbutils.NewDBConnect(connectionStringDB)
	if err != nil {
		logger.Fatal("Ошибка подключения к базе данных:", "err", err)
	}
	// БД информации пользователей.
	userStorage := db.NewUserStorage(dbconn, mainCurrency, 0)
	// БД курсов валют.
	exchangeRatesStorage := db.NewExchangeRatesStorage(dbconn, currenciesName)

	// Инициализация клиента загрузки курсов валют из внешнего источника.
	ctx, cancel := signal.NotifyContext(ctx,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer cancel()
	httpClient := net_http.New[cbr.ExchangeRatesJson]()
	cbrClient := cbr.New(ctx, httpClient)

	// Инициализация локального экземпляра класса для работы с курсами валют.
	exchangeRates := rates.New(ctx, cbrClient, currenciesName, mainCurrency, exchangeRatesStorage)

	// Инициализация кэша для кэширования отчетов пользователей.
	cacheLRU := cache.NewLRU(5)

	// Инициализация кафки для отправки сообщений в очередь.
	kafkaProducer, err := kafka.NewSyncProducer(brokersList, kafkaTopic)
	if err != nil {
		logger.Fatal("Ошибка инициализации кафки для отправки сообщений:", "err", err)
	}

	// Инициализация основной модели.
	msgModel := messages.New(ctx, tgClient, userStorage, exchangeRates, cacheLRU, kafkaProducer)

	// Запуск периодическое обновление курсов валют.
	uploader.ExchangeRatesUpdater(ctx, exchangeRates, currenciesUpdatePeriod)

	// Запуск периодическое обновление локального кэша курсов валют из БД.
	uploader.ExchangeRatesFromStorageLoader(ctx, exchangeRates, currenciesUpdateCachePeriod)

	// Запуск сервера для получения отчетов пользователя.
	reportserver.StartReportServer(msgModel)

	// Запуск ТГ-клиента.
	tgClient.ListenUpdates(msgModel)

	logger.Info("Завершение приложения")
}

// Замена параметров по умолчанию параметрами из конфиг.файла.
func setConfigSettings(config config.Config) {
	if config.MainCurrency != "" {
		mainCurrency = config.MainCurrency
	}
	if len(config.CurrenciesName) > 0 {
		currenciesName = config.CurrenciesName
	}
	if config.CurrenciesUpdatePeriod > 0 {
		currenciesUpdatePeriod = time.Duration(config.CurrenciesUpdatePeriod) * time.Minute
	}
	if config.CurrenciesUpdateCachePeriod > 0 {
		currenciesUpdateCachePeriod = time.Duration(config.CurrenciesUpdateCachePeriod) * time.Minute
	}
	if config.ConnectionStringDB != "" {
		connectionStringDB = config.ConnectionStringDB
	}
	if config.KafkaTopic != "" {
		kafkaTopic = config.KafkaTopic
	}
	if len(config.BrokersList) > 0 {
		brokersList = config.BrokersList
	}
}
