// Сервис генерации отчетов.
package main

import (
	"context"
	"github.com/pkg/errors"
	"github.com/ellavs/tg-bot-golang/internal/api"
	"github.com/ellavs/tg-bot-golang/internal/config"
	"github.com/ellavs/tg-bot-golang/internal/helpers/dbutils"
	"github.com/ellavs/tg-bot-golang/internal/helpers/kafka"
	"github.com/ellavs/tg-bot-golang/internal/logger"
	"github.com/ellavs/tg-bot-golang/internal/model/bottypes"
	"github.com/ellavs/tg-bot-golang/internal/model/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

// Параметры по умолчанию (могут быть изменены через config)
var (
	mainCurrency       = "RUB"                      // Основная валюта для хранения данных.
	connectionStringDB = ""                         // Строка подключения к базе данных.
	kafkaTopic         = "tgbot"                    // Наименование топика Kafka.
	brokersList        = []string{"localhost:9092"} // Список адресов брокеров сообщений (адрес Kafka).
)

func main() {

	logger.Info("[Report service] Старт приложения")

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	defer cancel()

	config, err := config.New()
	if err != nil {
		logger.Fatal("[Report service] Ошибка получения файла конфигурации:", "err", err)
	}

	// Изменение параметров по умолчанию из заданной конфигурации.
	setConfigSettings(config.GetConfig())

	// Инициализация хранилищ (подключение к базе данных).
	dbconn, err := dbutils.NewDBConnect(connectionStringDB)
	if err != nil {
		logger.Fatal("[Report service] Ошибка подключения к базе данных:", "err", err)
	}
	// БД информации пользователей.
	userStorage := db.NewUserStorage(dbconn, mainCurrency, 0)

	// Инициализация кафки для получения сообщений из очереди.
	kafkaConsumer, err := kafka.NewConsumer(ctx, brokersList, kafkaTopic)
	if err != nil {
		logger.Fatal("[Report service] Ошибка инициализации кафки:", "err", err)
	}

	// Назначение функции, которая будет обрабатывать входящие сообщения из кафки.
	handlerFunc := func(ctx context.Context, key string, value string) error {
		return getKafkaMessage(ctx, userStorage, key, value)
	}

	// Запуск чтения сообщений из очереди.
	if err := kafkaConsumer.RunConsume(handlerFunc); err != nil {
		logger.Fatal("[Report service] Ошибка чтения кафки:", "err", err)
	}

	<-ctx.Done()
	logger.Info("[Report service] Завершение приложения")
}

// setConfigSettings Замена параметров по умолчанию параметрами из конфиг.файла.
func setConfigSettings(config config.Config) {
	if config.MainCurrency != "" {
		mainCurrency = config.MainCurrency
	}
	if config.ConnectionStringDB != "" {
		connectionStringDB = config.ConnectionStringDB
	}
}

// getKafkaMessage Получение запроса на построение отчета пользователя из кафки.
func getKafkaMessage(ctx context.Context, userStorage *db.UserStorage, key string, value string) error {
	if key == "" || value == "" {
		logger.Error("[Report service] Сообщение кафка содержит пустой ключ или значение.")
		return nil
	}
	userID, err := strconv.Atoi(key)
	if err != nil {
		logger.Error("[Report service] Сообщение кафка содержит некорректный ключ.", "err", err)
		return nil
	}
	// Получение данных для отчета из БД.
	periodDate := time.Now()
	switch value {
	case "w":
		periodDate = periodDate.AddDate(0, 0, -7)
	case "m":
		periodDate = periodDate.AddDate(0, -1, 0)
	case "y":
		periodDate = periodDate.AddDate(-1, 0, 0)
	}
	dt, err := userStorage.GetUserDataRecord(ctx, int64(userID), periodDate)
	if err != nil {
		logger.Error("[Report service] Ошибка получения отчета.", "err", err)
		return err
	}
	// Вызов бота для отправки отчета пользователю.
	err = sendReportToBot(dt, int64(userID), value)
	if err != nil {
		logger.Error("[Report service] Ошибка отправки отчета.", "err", err)
		return err
	}
	return nil
}

// sendReportToBot Вызов сервиса бота, принимающего данные для отчета по gRPC
func sendReportToBot(dt []bottypes.UserDataReportRecord, userID int64, reportKey string) error {

	// Соединение с сервисом бота.
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := api.NewUserReportsReciverClient(conn)

	// Подготовка данных для отправки по gRPC.
	items := make([]*api.ReportItem, len(dt))
	for ind, r := range dt {
		items[ind] = &api.ReportItem{Category: r.Category, Sum: float32(r.Sum)}
	}

	// Отправка данных.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.PutReport(ctx, &api.ReportRequest{Items: items, UserID: userID, ReportKey: reportKey})
	if err != nil {
		logger.Error("Ошибка отправки", "err", err)
	}

	if r.Valid {
		return nil
	}
	return errors.New("Отчет не отправлен.")
}
