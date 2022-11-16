// Package db - Работа с хранилищами (базой данных).
package db

// Работа с хранилищем информации о пользователях.

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/ellavs/tg-bot-golang/internal/helpers/dbutils"
	"github.com/ellavs/tg-bot-golang/internal/helpers/timeutils"
	types "github.com/ellavs/tg-bot-golang/internal/model/bottypes"
)

// UserDataReportRecordDB - Тип, принимающий структуру записей о расходах.
type UserDataReportRecordDB struct {
	Category string `db:"name"`
	Sum      int64  `db:"sum"`
}

// UserStorage - Тип для хранилища информации о пользователях.
type UserStorage struct {
	db              *sqlx.DB
	defaultCurrency string
	defaultLimits   int64
}

// NewUserStorage - Инициализация хранилища информации о пользователях.
// db - *sqlx.DB - ссылка на подключение к БД.
// defaultCurrency - string - валюта по умолчанию.
// defaultLimits - int64 - бюджет по умолчанию.
func NewUserStorage(db *sqlx.DB, defaultCurrency string, defaultLimits int64) *UserStorage {
	return &UserStorage{
		db:              db,
		defaultCurrency: defaultCurrency,
		defaultLimits:   defaultLimits,
	}
}

// InsertUser Добавление пользователя в базу данных.
func (storage *UserStorage) InsertUser(ctx context.Context, userID int64, userName string) error {
	// Запрос на добавление данных.
	const sqlString = `
		INSERT INTO users (tg_id, name, currency, limits)
			VALUES ($1, $2, $3, $4)
			 ON CONFLICT (tg_id) DO NOTHING;`

	// Выполнение запроса на добавление данных.
	if _, err := dbutils.Exec(ctx, storage.db, sqlString, userID, userName, storage.defaultCurrency, storage.defaultLimits); err != nil {
		return err
	}
	return nil
}

// CheckIfUserExist Проверка существования пользователя в базе данных.
func (storage *UserStorage) CheckIfUserExist(ctx context.Context, userID int64) (bool, error) {
	// Запрос на выборку пользователя.
	const sqlString = `SELECT COUNT(id) AS countusers FROM users WHERE tg_id = $1;`

	// Выполнение запроса на получение данных.
	cnt, err := dbutils.GetMap(ctx, storage.db, sqlString, userID)
	if err != nil {
		return false, err
	}
	// Приведение результата запроса к нужному типу.
	countusers, ok := cnt["countusers"].(int64)
	if !ok {
		return false, errors.New("Ошибка приведения типа результата запроса.")
	}
	if countusers == 0 {
		return false, nil
	}
	return true, nil
}

// CheckIfUserExistAndAdd Проверка существования пользователя в базе данных добавление, если не существует.
func (storage *UserStorage) CheckIfUserExistAndAdd(ctx context.Context, userID int64, userName string) (bool, error) {
	exist, err := storage.CheckIfUserExist(ctx, userID)
	if err != nil {
		return false, err
	}
	if !exist {
		// Добавление пользователя в базу, если не существует.
		err := storage.InsertUser(ctx, userID, userName)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// InsertUserDataRecord Добавление записи о расходах пользователя (в транзакции с проверкой превышения лимита).
func (storage *UserStorage) InsertUserDataRecord(ctx context.Context, userID int64, rec types.UserDataRecord, userName string, limitPeriod time.Time) (bool, error) {
	// Проверка существования пользователя в БД.
	_, err := storage.CheckIfUserExistAndAdd(ctx, userID, userName)
	if err != nil {
		return false, err
	}

	// Проверка, что не превышен лимит расходов.
	isOverLimit, err := checkIfUserOverLimit(ctx, storage.db, userID, limitPeriod)
	if err != nil {
		return false, err
	}
	if isOverLimit {
		return true, nil
	}

	// Запуск транзакции.
	err = dbutils.RunTx(ctx, storage.db,
		// Функция, выполняемая внутри транзакции.
		// Если функция вернет ошибку, произойдет откат транзакции.
		func(tx *sqlx.Tx) error {
			isOverLimit, err = insertUserDataRecordTx(ctx, tx, userID, rec, limitPeriod)
			return err
		})

	// Функция возвращает признак isOverLimit:
	// - true - запись не добавлена из-за превышения лимита,
	// - false - запись добавлена (при err == nil).
	return isOverLimit, err
}

// GetUserDataRecord Получение информации о расходах по категориям за период.
func (storage *UserStorage) GetUserDataRecord(ctx context.Context, userID int64, period time.Time) ([]types.UserDataReportRecord, error) {
	// Отбор записей по пользователю за указанный период с группировкой по категориям.
	const sqlString = `
		SELECT c.name, SUM(r.sum)
		FROM usermoneytransactions AS r
				 INNER JOIN usercategories AS c
							ON r.category_id = c.id
				 INNER JOIN users AS u
							ON r.user_id = u.id
		WHERE u.tg_id = $1 AND r.period >= $2
		GROUP BY c.name
		ORDER BY c.name;`

	var recs []UserDataReportRecordDB
	// Выполнение запроса на выборку данных (запись в переменную recs).
	if err := dbutils.Select(ctx, storage.db, &recs, sqlString, userID, period); err != nil {
		return nil, errors.Wrap(err, "Get user data record error")
	}

	result := make([]types.UserDataReportRecord, len(recs))
	for ind, rec := range recs {
		result[ind] = types.UserDataReportRecord{
			Category: rec.Category,
			Sum:      float64(rec.Sum),
		}
	}
	return result, nil
}

// InsertCategory Добавление категории пользователя.
func (storage *UserStorage) InsertCategory(ctx context.Context, userID int64, catName string, userName string) error {
	// Проверка существования пользователя в БД.
	_, err := storage.CheckIfUserExistAndAdd(ctx, userID, userName)
	if err != nil {
		return err
	}
	// Обрезка до 30 символов для удобства дальнейших отчетов.
	if len(catName) > 30 {
		catName = string(catName[:30])
	}
	// Запрос на добавление данных.
	const sqlString = `
		INSERT INTO usercategories (user_id, name)
			(SELECT id, $1 FROM users WHERE users.tg_id = $2)
		ON CONFLICT (user_id, lower(name)) DO NOTHING;`

	// Выполнение запроса на добавление данных.
	if _, err := dbutils.Exec(ctx, storage.db, sqlString, catName, userID); err != nil {
		return err
	}
	return nil
}

// GetUserCategory Получение списка категорий пользователя.
func (storage *UserStorage) GetUserCategory(ctx context.Context, userID int64) ([]string, error) {
	// Отбор категорий по пользователю.
	const sqlString = `
		SELECT c.name
		FROM usercategories AS c
			INNER JOIN users AS u
				ON c.user_id = u.id
		WHERE u.tg_id = $1
		GROUP BY c.name
		ORDER BY c.name;`

	var recs []string
	// Выполнение запроса на выборку данных (запись в переменную recs).
	if err := dbutils.Select(ctx, storage.db, &recs, sqlString, userID); err != nil {
		return nil, errors.Wrap(err, "Get user category error")
	}
	return recs, nil
}

// GetUserCurrency Получение выбранной валюты пользователя.
func (storage *UserStorage) GetUserCurrency(ctx context.Context, userID int64) (string, error) {
	// Получение  валюты по пользователю.
	const sqlString = `SELECT currency FROM users WHERE tg_id = $1;`

	// Выполнение запроса на выборку данных (запись результата запроса в map).
	result, err := dbutils.GetMap(ctx, storage.db, sqlString, userID)
	if err != nil {
		return "", errors.Wrap(err, "Get user currency error")
	}
	// Приведение результата запроса к нужному типу.
	currency, ok := result["currency"].(string)
	if !ok {
		return "", errors.New("Ошибка приведения типа результата запроса.")
	}
	return currency, nil
}

// SetUserCurrency Сохранение выбранной валюты пользователя.
func (storage *UserStorage) SetUserCurrency(ctx context.Context, userID int64, currencyName string, userName string) error {
	// Проверка существования пользователя в БД.
	_, err := storage.CheckIfUserExistAndAdd(ctx, userID, userName)
	if err != nil {
		return err
	}
	// Запрос на обновление данных.
	const sqlString = `UPDATE users SET currency = $1 WHERE tg_id = $2;`

	// Выполнение запроса на обновление данных.
	if _, err := dbutils.Exec(ctx, storage.db, sqlString, currencyName, userID); err != nil {
		return err
	}
	return nil
}

// GetUserLimit Получение бюджета пользователя.
func (storage *UserStorage) GetUserLimit(ctx context.Context, userID int64) (int64, error) {
	// Получение бюджета по пользователю.
	const sqlString = `SELECT limits FROM users WHERE tg_id = $1;`

	// Выполнение запроса на выборку данных (запись результата запроса в map).
	result, err := dbutils.GetMap(ctx, storage.db, sqlString, userID)
	if err != nil {
		return 0, errors.Wrap(err, "Get user limits error")
	}
	// Приведение результата запроса к нужному типу.
	limits, ok := result["limits"].(int64)
	if !ok {
		return 0, errors.New("Ошибка приведения типа результата запроса.")
	}
	return limits, nil
}

// SetUserLimit Сохранение бюджета пользователя.
func (storage *UserStorage) SetUserLimit(ctx context.Context, userID int64, limits int64, userName string) error {
	// Проверка существования пользователя в БД.
	_, err := storage.CheckIfUserExistAndAdd(ctx, userID, userName)
	if err != nil {
		return err
	}
	// Запрос на обновление данных.
	const sqlString = `UPDATE users SET limits = $1 WHERE tg_id = $2;`

	// Выполнение запроса на обновление данных.
	if _, err := dbutils.Exec(ctx, storage.db, sqlString, limits, userID); err != nil {
		return err
	}
	return nil
}

// checkIfUserOverLimit Проверка, что текущие расходы пользователя не превзошли лимит (можно вызывать внутри транзакции).
func checkIfUserOverLimit(ctx context.Context, db sqlx.QueryerContext, userID int64, period time.Time) (bool, error) {
	// Проверка наличия записей о расходах.
	isRecsExist, err := checkIfUserRecordsExistInPeriod(ctx, db, userID, period)
	if err != nil {
		return false, err
	}
	if !isRecsExist {
		// Записей о расходах в указанном периоде нет, превышения нет.
		return false, nil
	}

	// Запрос на проверку лимита.
	const sqlString = `
		SELECT SUM(r.sum) > u.limits AND NOT u.limits = 0 AS overlimit
		FROM usermoneytransactions AS r
			INNER JOIN users AS u
				ON r.user_id = u.id
		WHERE u.tg_id = $1 AND r.period >= $2 AND r.period < $3
		GROUP BY u.limits;`

	// Выполнение запроса на получение данных.
	res, err := dbutils.GetMap(ctx, db, sqlString, userID, period, timeutils.BeginOfNextMonth(period))
	if err != nil {
		return false, err
	}
	// Приведение результата запроса к нужному типу.
	overlimit, ok := res["overlimit"].(bool)
	if !ok {
		return false, errors.New("Ошибка приведения типа результата запроса.")
	}
	if overlimit {
		return true, nil
	}
	return false, nil
}

// checkIfUserRecordsExistInPeriod Проверка наличия записей о расходах пользователя
// в базе данных (можно вызывать внутри транзакции).
func checkIfUserRecordsExistInPeriod(ctx context.Context, db sqlx.QueryerContext, userID int64, period time.Time) (bool, error) {
	// Запрос на проверку лимита.
	const sqlString = `
		SELECT COUNT(r.id) AS counter
		FROM users AS u
			INNER JOIN usermoneytransactions AS r
				ON r.user_id = u.id
		WHERE u.tg_id = $1 
		  AND r.period >= $2 AND r.period < $3
		;`

	// Выполнение запроса на получение данных.
	cnt, err := dbutils.GetMap(ctx, db, sqlString, userID, period, timeutils.BeginOfNextMonth(period))
	if err != nil {
		return false, err
	}
	// Приведение результата запроса к нужному типу.
	counter, ok := cnt["counter"].(int64)
	if !ok {
		return false, errors.New("Ошибка приведения типа результата запроса.")
	}
	if counter == 0 {
		return false, nil
	}
	return true, nil
}

// insertUserDataRecordTx Функция добавления расхода, выполняемая внутри транзакции (tx).
func insertUserDataRecordTx(ctx context.Context, tx sqlx.ExtContext, userID int64, rec types.UserDataRecord, limitPeriod time.Time) (bool, error) {

	// Запрос на добаление записи с проверкой существования категории.
	const sqlString = `
		WITH rows AS (INSERT INTO usercategories (user_id, name)
			(SELECT id, :category_name FROM users WHERE users.tg_id = :tg_id)
		ON CONFLICT (user_id, lower(name)) DO NOTHING)
		INSERT INTO usermoneytransactions (user_id, category_id, sum, period)
			(SELECT u.id, c.id, :sum, :period
			 FROM usercategories AS c
					  INNER JOIN users AS u ON c.user_id = u.id
			 WHERE u.tg_id = :tg_id AND lower(c.name) = lower(:category_name))
		ON CONFLICT DO NOTHING;`

	// Именованные параметры запроса.
	args := map[string]any{
		"tg_id":         userID,
		"category_name": rec.Category,
		"sum":           rec.Sum,
		"period":        rec.Period,
	}

	// Запуск на выполнение запроса с именованными параметрами.
	if _, err := dbutils.NamedExec(ctx, tx, sqlString, args); err != nil {
		// Ошибка выполнения запроса (вызовет откат транзакции).
		return false, err
	}

	// Проверка превышения (для отката транзакции в случае превышения).
	isOverLimit, err := checkIfUserOverLimit(ctx, tx, userID, limitPeriod)
	if err != nil {
		// Ошибка чтения из базы (вызовет откат транзакции).
		return false, err
	}
	if isOverLimit {
		// Признак превышения лимита (вызовет откат транзакции)
		return true, errors.New("Превышение лимита.")
	}
	// Возвращается признак, что превышения не было, ошибки отсутствуют
	// (вызовет коммит транзакции).
	return false, nil
}
