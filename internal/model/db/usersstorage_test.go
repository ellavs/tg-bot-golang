package db

import (
	"context"
	sqlxmock "github.com/zhashkevych/go-sqlxmock"
	"regexp"
	"testing"
)

func Test_UserStorage_InsertUser(t *testing.T) {

	ctx := context.Background()
	db, mock, err := sqlxmock.Newx()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	s := NewUserStorage(db, "RUB", 10000)

	tests := []struct {
		name     string
		s        *UserStorage
		userID   int64
		userName string
		mock     func()
		want     int64
		wantErr  bool
	}{
		{
			name:     "Должно быть без ошибок",
			s:        s,
			userID:   15236,
			userName: "test user name",
			mock: func() {
				mock.ExpectExec("INSERT INTO users").
					WithArgs(15236, "test user name", "RUB", 10000).WillReturnResult(sqlxmock.NewResult(0, 0))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			err := tt.s.InsertUser(ctx, tt.userID, tt.userName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Не совпало ожидание ошибки: Get() error new = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_UserStorage_CheckIfUserExist(t *testing.T) {

	ctx := context.Background()
	db, mock, err := sqlxmock.Newx()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	s := NewUserStorage(db, "RUB", 10000)

	tests := []struct {
		name    string
		s       *UserStorage
		userID  int64
		mock    func()
		want    bool
		wantErr bool
	}{
		{
			name:   "Тест 1. Должно быть без ошибок (пользователь существует).",
			s:      s,
			userID: 15236,
			mock: func() {
				rows := sqlxmock.NewRows([]string{"countusers"}).AddRow(1)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(id) AS countusers FROM users WHERE tg_id = $1;")).
					WithArgs(15236).WillReturnRows(rows)
			},
			wantErr: false,
			want:    true,
		},
		{
			name:   "Тест 2. Должно быть без ошибок (пользователь не существует).",
			s:      s,
			userID: 15237,
			mock: func() {
				rows := sqlxmock.NewRows([]string{"countusers"}).AddRow(0)
				mock.ExpectQuery(regexp.QuoteMeta("SELECT COUNT(id) AS countusers FROM users WHERE tg_id = $1;")).
					WithArgs(15237).WillReturnRows(rows)
			},
			wantErr: false,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mock()
			got, err := tt.s.CheckIfUserExist(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Не совпало ожидание ошибки: Get() error new = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && got != tt.want {
				t.Errorf("Не совпало ожидание получаемого значения: Get() = %v, want %v", got, tt.want)
			}
		})
	}
}
