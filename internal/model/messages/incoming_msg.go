package messages

import (
	"context"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"github.com/ellavs/tg-bot-golang/internal/logger"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/ellavs/tg-bot-golang/internal/helpers/timeutils"
	types "github.com/ellavs/tg-bot-golang/internal/model/bottypes"
)

// Область "Константы и переменные": начало.

const (
	txtStart            = "Привет, *%v*. Я помогаю вести учет расходов. Выберите действие."
	txtUnknownCommand   = "К сожалению, данная команда мне неизвестна. Для начала работы введите /start"
	txtReportError      = "Не удалось получить данные."
	txtReportEmpty      = "За указанный период данные отсутствуют."
	txtReportWait       = "Формирование отчета. Пожалуйста, подождите..."
	txtCatAdd           = "Введите название категории (не более 30 символов). Для отмены введите 0."
	txtCatView          = "Выберите категорию, а затем введите сумму."
	txtCatChoice        = "Выбрана категория *%v*. Введите сумму (только число). Для отмены введите 0. Используемая валюта: *%v*"
	txtCatSave          = "Категория успешно сохранена."
	txtCatEmpty         = "Пока нет категорий, сначала добавьте хотя бы одну категорию."
	txtRecSave          = "Запись успешно сохранена."
	txtRecOverLimit     = "Запись не сохранена: превышен бюджет раходов в текущем месяце."
	txtRecTbl           = "Для загрузки истории расходов введите таблицу в следующем формате (дата сумма категория):\n`YYYY-MM-DD 0.00 XXX`\nНапример: \n`2022-09-20 1500 Кино`\n`2022-07-12 350.50 Продукты, еда`\n`2022-08-30 8000 Одежда и обувь`\n`2022-09-01 60 Бензин`\n`2022-09-27 425 Такси`\n`2022-09-26 1500 Бензин`\n`2022-09-26 950 Кошка`\n`2022-09-25 50 Бензин`\nИспользуемая валюта: *%v*"
	txtReportQP         = "За какой период будем смотреть отчет? Команды периодов: /report_w - неделя, /report_m - месяц, /report_y - год"
	txtHelp             = "Я - бот, помогающий вести учет расходов. Для начала работы введите /start"
	txtCurrencyChoice   = "В качестве основной задана валюта: *%v*. Для изменения выберите другую валюту."
	txtCurrencySet      = "Валюта изменена на *%v*."
	txtCurrencySetError = "Ошибка сохранения валюты."
	txtLimitInfo        = "Текущий ежемесячный бюджет: *%v*. Для изменения введите число, например, 80000."
	txtLimitSet         = "Бюджет изменен на *%v*."
)

// Команды стартовых действий.
var btnStart = []types.TgRowButtons{
	{types.TgInlineButton{DisplayName: "Добавить категорию", Value: "/add_cat"}, types.TgInlineButton{DisplayName: "Добавить расход", Value: "/add_rec"}},
	{types.TgInlineButton{DisplayName: "Отчет за неделю", Value: "/report_w"}, types.TgInlineButton{DisplayName: "Отчет за месяц", Value: "/report_m"}, types.TgInlineButton{DisplayName: "Отчет за год", Value: "/report_y"}},
	{types.TgInlineButton{DisplayName: "Ввести данные за прошлый период", Value: "/add_tbl"}},
	{types.TgInlineButton{DisplayName: "Выбрать валюту", Value: "/choice_currency"}, types.TgInlineButton{DisplayName: "Установить бюджет", Value: "/set_limit"}},
}

var lineRegexp = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}) (\d+.?\d{0,2}) (.+)$`)

// Область "Константы и переменные": конец.

// Область "Внешний интерфейс": начало.

// MessageSender Интерфейс для работы с сообщениями.
type MessageSender interface {
	SendMessage(text string, userID int64) error
	ShowInlineButtons(text string, buttons []types.TgRowButtons, userID int64) error
}

// UserDataStorage Интерфейс для работы с хранилищем данных.
type UserDataStorage interface {
	InsertUserDataRecord(ctx context.Context, userID int64, rec types.UserDataRecord, userName string, limitPeriod time.Time) (bool, error)
	GetUserDataRecord(ctx context.Context, userID int64, period time.Time) ([]types.UserDataReportRecord, error)
	InsertCategory(ctx context.Context, userID int64, catName string, userName string) error
	GetUserCategory(ctx context.Context, userID int64) ([]string, error)
	GetUserCurrency(ctx context.Context, userID int64) (string, error)
	SetUserCurrency(ctx context.Context, userID int64, currencyName string, userName string) error
	GetUserLimit(ctx context.Context, userID int64) (int64, error)
	SetUserLimit(ctx context.Context, userID int64, limits int64, userName string) error
}

// ExchangeRates Интерфейс для работы с курсами валют.
type ExchangeRates interface {
	ConvertSumFromBaseToCurrency(currencyName string, sum int64) (int64, error)
	ConvertSumFromCurrencyToBase(currencyName string, sum int64) (int64, error)
	GetExchangeRate(currencyName string) (float64, error)
	GetMainCurrency() string
	GetCurrenciesList() []string
}

// LRUCache Интерфейс для работы с кэшем отчетов.
type LRUCache interface {
	Add(key string, value any)
	Get(key string) any
}

// kafkaProducer Интерфейс для отправки сообщений в кафку.
type kafkaProducer interface {
	SendMessage(key string, value string) (partition int32, offset int64, err error)
	GetTopic() string
}

// Model Модель бота (клиент, хранилище, последние команды пользователя)
type Model struct {
	ctx             context.Context
	tgClient        MessageSender    // Клиент.
	storage         UserDataStorage  // Хранилище пользовательской информации.
	currencies      ExchangeRates    // Хранилише курсов валют.
	reportCache     LRUCache         // Хранилише кэша.
	kafkaProducer   kafkaProducer    // Кафка
	lastUserCat     map[int64]string // Последняя выбранная пользователем категория.
	lastUserCommand map[int64]string // Последняя выбранная пользователем команда.
}

// New Генерация сущности для хранения клиента ТГ и хранилища пользователей и курсов валют.
func New(ctx context.Context, tgClient MessageSender, storage UserDataStorage, currencies ExchangeRates, reportCache LRUCache, kafka kafkaProducer) *Model {
	return &Model{
		ctx:             ctx,
		tgClient:        tgClient,
		storage:         storage,
		lastUserCat:     map[int64]string{},
		lastUserCommand: map[int64]string{},
		currencies:      currencies,
		reportCache:     reportCache,
		kafkaProducer:   kafka,
	}
}

// Message Структура сообщения для обработки.
type Message struct {
	Text            string
	UserID          int64
	UserName        string
	UserDisplayName string
	IsCallback      bool
	CallbackMsgID   string
}

func (s *Model) GetCtx() context.Context {
	return s.ctx
}

func (s *Model) SetCtx(ctx context.Context) {
	s.ctx = ctx
}

// IncomingMessage Обработка входящего сообщения.
func (s *Model) IncomingMessage(msg Message) error {
	span, ctx := opentracing.StartSpanFromContext(s.ctx, "IncomingMessage")
	s.ctx = ctx
	defer span.Finish()

	lastUserCat := s.lastUserCat[msg.UserID]
	lastUserCommand := s.lastUserCommand[msg.UserID]

	// Обнуление выбранной категории и команды.
	s.lastUserCat[msg.UserID] = ""
	s.lastUserCommand[msg.UserID] = ""

	// Проверка ввода суммы расхода по выбранной категории и сохранение, если введено.
	if isNeedReturn, err := checkIfEnterCategorySum(s, msg, lastUserCat); err != nil || isNeedReturn {
		return err
	}

	// Проверка ввода новой категории и сохранение, если введено.
	if isNeedReturn, err := checkIfEnterNewCategory(s, msg, lastUserCommand); err != nil || isNeedReturn {
		return err
	}

	// Проверка ввода бюджета и сохранение, если введено.
	if isNeedReturn, err := checkIfEnterNewLimit(s, msg, lastUserCommand); err != nil || isNeedReturn {
		return err
	}

	// Проверка ввода данных в виде таблицы и сохранение, если введено.
	if isNeedReturn, err := checkIfEnterTableData(s, msg, lastUserCommand); err != nil || isNeedReturn {
		return err
	}

	// Проверка выбора категории для ввода расхода.
	if isNeedReturn, err := checkIfCoiceCategory(s, msg); err != nil || isNeedReturn {
		return err
	}

	// Проверка выбора валюты.
	if isNeedReturn, err := checkIfCoiceCurrency(s, msg); err != nil || isNeedReturn {
		return err
	}

	// Распознавание стандартных команд.
	if isNeedReturn, err := checkBotCommands(s, msg); err != nil || isNeedReturn {
		return err
	}

	// Отправка ответа по умолчанию.
	return s.tgClient.SendMessage(txtUnknownCommand, msg.UserID)
}

// SendReportToUser Отправка отчета за период.
func (s *Model) SendReportToUser(dt []types.UserDataReportRecord, userID int64, reportKey string) error {
	span, ctx := opentracing.StartSpanFromContext(s.ctx, "SendReportToUser")
	s.ctx = ctx
	defer span.Finish()

	strReportTitle := "Отчет за "
	switch reportKey {
	case "w":
		strReportTitle += "*последнюю неделю*"
	case "m":
		strReportTitle += "*последний месяц*"
	case "y":
		strReportTitle += "*последний год*"
	}

	// Получение данных из БД.
	userCurrency := getUserCurrency(s, userID)
	answerText := formatReport(s, dt, userCurrency)
	if len(answerText) == 0 {
		answerText = txtReportEmpty
	} else {
		answerText = fmt.Sprintln(strReportTitle+" ("+userCurrency+")") + answerText
	}
	// Сохранение значения в кэш.
	reportCacheKey := strconv.Itoa(int(userID)) + reportKey
	s.reportCache.Add(reportCacheKey, answerText)
	err := s.tgClient.SendMessage(answerText, userID)
	if err != nil {
		logger.Error("Ошибка отправки сообщения в ТГ", "err", err)
		return err
	}
	return nil
}

// Область "Внешний интерфейс": конец.

// Область "Служебные функции": начало.

// Область "Распознавание входящих команд": начало.

// Проверка ввода суммы расхода по выбранной категории.
func checkIfEnterCategorySum(s *Model, msg Message, lastUserCat string) (bool, error) {
	// Если выбрана категория и введена сумма, то сохранение записи о расходах.
	if lastUserCat != "" && msg.Text != "" {
		span, ctx := opentracing.StartSpanFromContext(s.ctx, "checkIfEnterCategorySum")
		s.ctx = ctx
		defer span.Finish()

		// Парсинг и конвертация введенной суммы.
		catSum, err := parseAndConvertSumFromCurrency(s, msg.UserID, msg.Text)
		if err != nil {
			return true, err
		}
		// Сохранение записи.
		newRec := types.UserDataRecord{UserID: msg.UserID, Category: lastUserCat, Sum: catSum, Period: time.Now()}
		isOverLimit, err := s.storage.InsertUserDataRecord(s.ctx, msg.UserID, newRec, msg.UserName, timeutils.BeginOfMonth(newRec.Period))
		if err != nil {
			if isOverLimit {
				// Было превышение лимита.
				return true, s.tgClient.SendMessage(txtRecOverLimit, msg.UserID)
			} else {
				logger.Error("Ошибка сохранения записи", "err", err)
				return true, errors.Wrap(err, "Insert data record error")
			}
		}
		// Ответ пользователю об успешном сохранении.
		return true, s.tgClient.SendMessage(txtRecSave, msg.UserID)
	}
	// Это не ввод расхода.
	return false, nil
}

// Проверка ввода новой категории и сохранение, если введено.
func checkIfEnterNewCategory(s *Model, msg Message, lastUserCommand string) (bool, error) {
	if lastUserCommand == "/add_cat" {
		span, ctx := opentracing.StartSpanFromContext(s.ctx, "checkIfEnterNewCategory")
		s.ctx = ctx
		defer span.Finish()

		// Выбрано добавление категории.
		if msg.Text == "0" {
			// Ввод категории отменен.
			return true, nil
		} else {
			// Сохранение категории.
			err := s.storage.InsertCategory(s.ctx, msg.UserID, msg.Text, msg.UserName)
			if err != nil {
				logger.Error("Ошибка сохранения категории", "err", err)
				return true, errors.Wrap(err, "Insert category error")
			}
			// Ответ пользователю об успешном сохранении.
			return true, s.tgClient.SendMessage(txtCatSave, msg.UserID)
		}
	}
	// Это не ввод новой категории.
	return false, nil
}

// Проверка ввода бюджета и сохранение, если введено.
func checkIfEnterNewLimit(s *Model, msg Message, lastUserCommand string) (bool, error) {
	// Если выбрано добавление бюджета и введена сумма, то сохранение.
	if lastUserCommand == "/set_limit" && msg.Text != "" {
		span, ctx := opentracing.StartSpanFromContext(s.ctx, "checkIfEnterNewLimit")
		s.ctx = ctx
		defer span.Finish()

		// Парсинг и конвертация введенной суммы.
		limit, err := parseAndConvertSumFromCurrency(s, msg.UserID, msg.Text)
		if err != nil {
			return true, err
		}
		if limit >= 0 {
			// Сохранение бюджета.
			err = s.storage.SetUserLimit(s.ctx, msg.UserID, limit, msg.UserName)
			if err != nil {
				logger.Error("Ошибка сохранения бюджета", "err", err)
				return true, errors.Wrap(err, "Ошибка сохранения бюджета.")
			}
			// Ответ пользователю об успешном сохранении.
			return true, s.tgClient.SendMessage(fmt.Sprintf(txtLimitSet, msg.Text), msg.UserID)
		}
	}
	// Это не ввод бюджета.
	return false, nil
}

// Проверка ввода данных в виде таблицы и сохранение, если введено.
func checkIfEnterTableData(s *Model, msg Message, lastUserCommand string) (bool, error) {
	if lastUserCommand == "/add_tbl" {
		span, ctx := opentracing.StartSpanFromContext(s.ctx, "checkIfEnterTableData")
		s.ctx = ctx
		defer span.Finish()

		// Выбрано добавление таблиных данных.
		answerText := ""
		if msg.Text == "0" {
			// Ввод отменен.
			return true, nil
		} else {
			// Парсинг данных.
			lines := strings.Split(msg.Text, "\n")

			for ind, line := range lines {
				isError := false
				txtError := ""
				rec, err := parseLineRec(line)
				if err != nil {
					isError = true
					txtError = "Ошибка распознавания формата строки."
				} else {
					// Сохранение данных.
					if err := s.storage.InsertCategory(s.ctx, msg.UserID, rec.Category, msg.UserName); err != nil {
						isError = true
						txtError = "Ошибка добавления категории."
					} else {
						rec.UserID = msg.UserID
						// Конвертация из валюты пользователя в базовую.
						if sum, err := convertSumFromCurrency(s, msg.UserID, rec.Sum); err != nil {
							isError = true
							txtError = "Ошибка конвертации валюты."
						} else {
							rec.Sum = sum
							// Сохранение записи.
							if isOverLimit, err := s.storage.InsertUserDataRecord(s.ctx, msg.UserID, rec, msg.UserName, timeutils.BeginOfMonth(rec.Period)); err != nil {
								isError = true
								if isOverLimit {
									txtError = "Превышение бюджета."
								} else {
									txtError = "Ошибка сохранения записи."
								}
							}
						}
					}
				}
				if isError {
					answerText += fmt.Sprintf("%v. Ошибка. %v \n", ind+1, txtError)
				} else {
					answerText += fmt.Sprintf("%v. ОК\n", ind+1)
				}
			}
			// Ответ пользователю об сохранении.
			answerText = txtRecSave + "\n" + answerText
			return true, s.tgClient.SendMessage(answerText, msg.UserID)
		}
	}
	// Это не ввод таблицы.
	return false, nil
}

// Проверка выбора категории для ввода расхода.
func checkIfCoiceCategory(s *Model, msg Message) (bool, error) {
	// Распознавание нажатых кнопок выбора категорий.
	if msg.IsCallback {
		if strings.Contains(msg.Text, "/cat ") {
			span, ctx := opentracing.StartSpanFromContext(s.ctx, "checkIfCoiceCategory")
			s.ctx = ctx
			defer span.Finish()

			// Пользователь выбрал категорию.
			cat := strings.Replace(msg.Text, "/cat ", "", -1)
			answerText := fmt.Sprintf(txtCatChoice, cat, getUserCurrency(s, msg.UserID))
			s.lastUserCat[msg.UserID] = cat
			return true, s.tgClient.SendMessage(answerText, msg.UserID)
		}
	}
	// Это не выбор категории.
	return false, nil
}

// Проверка выбора валюты.
func checkIfCoiceCurrency(s *Model, msg Message) (bool, error) {
	// Распознавание нажатых кнопок выбора валюты.
	if msg.IsCallback {
		if strings.Contains(msg.Text, "/curr ") {
			span, ctx := opentracing.StartSpanFromContext(s.ctx, "checkIfCoiceCurrency")
			s.ctx = ctx
			defer span.Finish()

			// Пользователь выбрал валюту.
			choice := strings.Replace(msg.Text, "/curr ", "", -1)
			answerText := fmt.Sprintf(txtCurrencySet, choice)
			// Сохранение выбранной валюты.
			if err := s.storage.SetUserCurrency(s.ctx, msg.UserID, choice, msg.UserName); err != nil {
				return true, s.tgClient.SendMessage(txtCurrencySetError, msg.UserID)
			} else {
				return true, s.tgClient.SendMessage(answerText, msg.UserID)
			}
		}
	}
	// Это не выбор категории.
	return false, nil
}

// Распознавание стандартных команд бота.
func checkBotCommands(s *Model, msg Message) (bool, error) {
	span, ctx := opentracing.StartSpanFromContext(s.ctx, "checkBotCommands")
	s.ctx = ctx
	defer span.Finish()

	switch msg.Text {
	case "/start":
		displayName := msg.UserDisplayName
		if len(displayName) == 0 {
			displayName = msg.UserName
		}
		// Отображение команд стартовых действий.
		return true, s.tgClient.ShowInlineButtons(fmt.Sprintf(txtStart, displayName), btnStart, msg.UserID)
	case "/report":
		return true, s.tgClient.SendMessage(txtReportQP, msg.UserID)
	case "/help":
		return true, s.tgClient.SendMessage(txtHelp, msg.UserID)
	case "/add_tbl":
		s.lastUserCommand[msg.UserID] = "/add_tbl"
		userCurrency := getUserCurrency(s, msg.UserID)
		return true, s.tgClient.SendMessage(fmt.Sprintf(txtRecTbl, userCurrency), msg.UserID)
	case "/report_w", "/report_m", "/report_y":
		// Отображение отчета.
		return true, s.tgClient.SendMessage(getReportByPeriod(s, msg), msg.UserID)
	case "/add_cat":
		// Отображение сообщения о вводе категории.
		s.lastUserCommand[msg.UserID] = "/add_cat"
		return true, s.tgClient.SendMessage(txtCatAdd, msg.UserID)
	case "/add_rec":
		s.lastUserCommand[msg.UserID] = "/add_rec"
		// Отображение кнопок с существующими категориями для выбора.
		if btnCat, err := getCategoryButtons(s, msg.UserID); err != nil || btnCat == nil {
			return true, err
		} else {
			return true, s.tgClient.ShowInlineButtons(txtCatView, btnCat, msg.UserID)
		}
	case "/choice_currency":
		// Отображение кнопок выбора валюты.
		userCurrency := getUserCurrency(s, msg.UserID)
		if btnCurr, err := getCurrencyButtons(s, userCurrency); err != nil {
			return true, err
		} else {
			return true, s.tgClient.ShowInlineButtons(fmt.Sprintf(txtCurrencyChoice, userCurrency), btnCurr, msg.UserID)
		}
	case "/set_limit":
		// Отображение сообщения о вводе бюджета.
		s.lastUserCommand[msg.UserID] = "/set_limit"
		answerText := fmt.Sprintf(txtLimitInfo, "без ограничений")
		userLimit, _ := getUserLimit(s, msg.UserID)
		if userLimit > 0 {
			answerText = fmt.Sprintf(txtLimitInfo, userLimit/100)
		}
		return true, s.tgClient.SendMessage(answerText, msg.UserID)
	}
	// Команда не распознана.
	return false, nil
}

// Область "Распознавание входящих команд": конец.

// Область "Формирование отчета": начало.

// Получение отчета за период.
func getReportByPeriod(s *Model, msg Message) string {
	span, ctx := opentracing.StartSpanFromContext(s.ctx, "getReportByPeriod")
	s.ctx = ctx
	defer span.Finish()

	answerText := ""
	reportKey := strings.Replace(msg.Text, "/report_", "", -1)

	// Ключ для поиска в кэше.
	reportCacheKey := strconv.Itoa(int(msg.UserID)) + reportKey
	// Попытка получить значение из кэша.
	cacheValue := s.reportCache.Get(reportCacheKey)
	if cacheValue != nil {
		answerText, ok := cacheValue.(string)
		if ok {
			return answerText
		} else {
			logger.Error("Ошибка приведения значения кэша к строке.")
		}
	}

	// Отправка запроса на формирование отчета в кафку.
	p, o, err := s.kafkaProducer.SendMessage(strconv.Itoa(int(msg.UserID)), reportKey)
	if err != nil {
		logger.Error("Ошибка отправки сообщения в кафку", "err", err)
		answerText = txtReportError
	} else {
		logger.Debug(fmt.Sprintf("[KAFKA] Successful to write message, topic %s, offset:%d, partition: %d\n", s.kafkaProducer.GetTopic(), o, p))
		answerText = txtReportWait
	}

	return answerText
}

// Форматирование массива данных в строку для вывода отчета.
func formatReport(s *Model, recs []types.UserDataReportRecord, userCurrency string) string {
	var res strings.Builder
	totalSum := 0.0
	for ind, rec := range recs {
		// Конвертация сумм в валюту пользователя.
		sumCurrency, err := s.currencies.ConvertSumFromBaseToCurrency(userCurrency, int64(rec.Sum))
		if err != nil {
			logger.Error("Ошибка конвертации валюты", "err", err)
			return "Ошибка конвертации валюты"
		}
		recs[ind].Sum = float64(sumCurrency) / 100
		totalSum += float64(sumCurrency) / 100
	}
	maxSumStr := fmt.Sprintf("%.2f", totalSum)

	res.WriteString(fmt.Sprintf("`%*s | %v`", len(maxSumStr)+1, "Сумма", "Категория") + "\n")
	res.WriteString(fmt.Sprintf("`%v`", strings.Repeat("-", len(maxSumStr)+15)) + "\n")

	for _, rec := range recs {
		// Форматирование категории и числа до нужной ширины.
		res.WriteString(fmt.Sprintf("`%*.2f | %v`", len(maxSumStr)+1, rec.Sum, rec.Category) + "\n")
	}
	if len(recs) > 0 {
		res.WriteString(fmt.Sprintf("`%v`", strings.Repeat("-", len(maxSumStr)+15)) + "\n")
		res.WriteString(fmt.Sprintf("`%*.2f | %v`", len(maxSumStr)+1, totalSum, "ИТОГО") + "\n")
	}
	return res.String()
}

// Область "Формирование отчета": конец.

// Область "Получение данных пользователя": начало.

// Получение кнопок с существующими категориями для выбора.
func getCategoryButtons(s *Model, userID int64) ([]types.TgRowButtons, error) {
	userCategories, err := s.storage.GetUserCategory(s.ctx, userID)
	if err != nil {
		logger.Error("Ошибка получения категорий", "err", err)
		return nil, errors.Wrap(err, "Get user categories error")
	}
	if len(userCategories) == 0 {
		return nil, s.tgClient.SendMessage(txtCatEmpty, userID)
	}
	var btnCat = []types.TgRowButtons{}
	rowCounter := 0
	btnCat = append(btnCat, types.TgRowButtons{})
	for ind, cat := range userCategories {
		if ind%3 == 0 && ind > 0 {
			rowCounter++
			btnCat = append(btnCat, types.TgRowButtons{})
		}
		btnCat[rowCounter] = append(btnCat[rowCounter], types.TgInlineButton{DisplayName: cat, Value: "/cat " + cat})
	}
	return btnCat, nil
}

// Получение кнопок с валютами для выбора.
func getCurrencyButtons(s *Model, userCurrency string) ([]types.TgRowButtons, error) {
	// Получение списка используемых валют.
	currenciesName := s.currencies.GetCurrenciesList()
	var buttons = make([]types.TgRowButtons, 1)
	for _, name := range currenciesName {
		// Добавление кнопок валют, кроме выбранной ранее пользователем.
		if name != userCurrency {
			buttons[0] = append(buttons[0], types.TgInlineButton{DisplayName: name, Value: "/curr " + name})
		}
	}
	return buttons, nil
}

// Получение выбранной валюты пользователя.
func getUserCurrency(s *Model, userID int64) string {
	userCurrency, _ := s.storage.GetUserCurrency(s.ctx, userID)
	if userCurrency == "" {
		// Если не задано, то - основная валюта.
		userCurrency = s.currencies.GetMainCurrency()
	}
	return userCurrency
}

// Получение бюджета пользователя.
func getUserLimit(s *Model, userID int64) (int64, error) {
	userLimit, err := s.storage.GetUserLimit(s.ctx, userID)
	if err != nil {
		logger.Error("Ошибка получения бюджета", "err", err)
		return 0, err
	}
	return userLimit, nil
}

// Область "Получение данных пользователя": конец.

// Область "Другие функции": начало.

// Парсинг строки данных.
func parseLineRec(line string) (types.UserDataRecord, error) {
	matches := lineRegexp.FindStringSubmatch(line)
	if len(matches) < 4 {
		return types.UserDataRecord{}, errors.New("Неверный формат строки.")
	}

	periodStr := matches[1]
	amountStr := matches[2]
	category := matches[3]

	// Парсинг числа.
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return types.UserDataRecord{}, errors.Wrap(err, "Некорректная сумма.")
	}

	// Парсинг даты.
	period, err := time.Parse("2006-01-02", periodStr)
	if err != nil {
		return types.UserDataRecord{}, errors.Wrap(err, "Некорректная дата.")
	}

	return types.UserDataRecord{
		Category: category,
		Sum:      int64(amount * 100),
		Period:   period,
	}, nil
}

// Конвертация суммы в базовую валюту для хранения.
func convertSumFromCurrency(s *Model, userID int64, sum int64) (int64, error) {
	userCurrency := getUserCurrency(s, userID)
	sumBase, err := s.currencies.ConvertSumFromCurrencyToBase(userCurrency, sum)
	if err != nil {
		logger.Error("Ошибка конвертации валюты", "err", err)
		return 0, err
	}
	return sumBase, nil
}

// Парсинг вводимого пользователем числа и конвертация суммы в базовую валюту.
func parseAndConvertSumFromCurrency(s *Model, userID int64, sumString string) (int64, error) {
	// Парсинг числа.
	sum, err := strconv.ParseFloat(sumString, 64)
	if err != nil {
		return 0, errors.Wrap(err, "Error parse sum")
	}
	// Сумма в разменных денежных единицах (1/100, для рублей - копейки, для долларов - центы и т.п.)
	nominalAmount := int64(sum * 100)
	// Конвертация из валюты пользователя в базовую валюту.
	if nominalAmount, err = convertSumFromCurrency(s, userID, nominalAmount); err != nil {
		return 0, errors.Wrap(err, "Ошибка конвертации валюты.")
	}
	return nominalAmount, nil
}

// Область "Другие функции": конец.

// Область "Служебные функции": конец.
