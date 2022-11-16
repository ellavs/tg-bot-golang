package metrics

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/ellavs/tg-bot-golang/internal/clients/tg"
	"github.com/ellavs/tg-bot-golang/internal/logger"
	"github.com/ellavs/tg-bot-golang/internal/model/messages"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type TgHandler interface {
	RunFunc(tgUpdate tgbotapi.Update, c *tg.Client, msgModel *messages.Model)
}

// Метрики.
var (
	InFlightRequests = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "tg",
		Subsystem: "messages",
		Name:      "messages_total", // Общее количество сообщений.
	})
	SummaryResponseTime = promauto.NewSummary(prometheus.SummaryOpts{
		Namespace: "tg",
		Subsystem: "messages",
		Name:      "summary_response_time_seconds", // Время обработки сообщений.
		Objectives: map[float64]float64{
			0.5:  0.1,
			0.9:  0.01,
			0.99: 0.001,
		},
	})
	HistogramResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "tg",
			Subsystem: "messages",
			Name:      "histogram_response_time_seconds",
			Buckets:   []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2},
		},
		[]string{"cmd"},
	)
)

var labels []string

func init() {
	labels = []string{"start", "cat", "curr", "report", "add_tbl", "add_cat", "add_rec", "choice_currency", "set_limit"}
	// Для просмотра значений метрик по адресу http://127.0.0.1:8080/
	http.Handle("/", promhttp.Handler())
	logger.Info("Старт сервиса метрик.")
	go func() {
		err := http.ListenAndServe("0.0.0.0:8080", nil)
		if err != nil {
			logger.Error("Metrics public error", "err", err)
		}
	}()
}

// MetricsMiddleware Функция сбора метрик.
func MetricsMiddleware(next tg.HandlerFunc) tg.HandlerFunc {

	handler := tg.HandlerFunc(func(tgUpdate tgbotapi.Update, c *tg.Client, msgModel *messages.Model) {

		// Сохранение времени начала обработки сообщения.
		startTime := time.Now()
		// Выполнение процесса обработки сообщения.
		next.RunFunc(tgUpdate, c, msgModel)
		// Расчет продолжительности обработки сообщения.
		duration := time.Since(startTime)

		// Сохранение метрик продолжительности обработки.
		SummaryResponseTime.Observe(duration.Seconds())

		// Определение команды для сохранения в метрике.
		cmd := "none"
		msg := ""
		if tgUpdate.Message == nil {
			if tgUpdate.CallbackQuery != nil {
				msg = tgUpdate.CallbackQuery.Data
			}
		} else {
			msg = tgUpdate.Message.Text
		}
		if msg != "" {
			for _, lbl := range labels {
				if strings.Contains(msg, "/"+lbl) {
					cmd = lbl
					break
				}
			}
		}
		HistogramResponseTime.
			WithLabelValues(cmd).
			Observe(duration.Seconds())

	})
	// Подсчет количества сообщений.
	InFlightRequests.Dec()

	return handler
}
