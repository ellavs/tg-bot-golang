package messages

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	mocks "github.com/ellavs/tg-bot-golang/internal/mocks/messages"
	types "github.com/ellavs/tg-bot-golang/internal/model/bottypes"
)

func Test_OnStartCommand_ShouldAnswerWithIntroMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)
	// Ожидаем ответ в виде сообщения c именем пользователя и кнопок меню.
	sender.EXPECT().ShowInlineButtons(fmt.Sprintf(txtStart, "Test"), btnStart, int64(123))

	// Запускаем тест модели - команда старт
	model := New(context.Background(), sender, nil, nil, nil, nil)
	err := model.IncomingMessage(Message{
		Text:            "/start",
		UserID:          123,
		UserName:        "test",
		UserDisplayName: "Test",
	})

	assert.NoError(t, err)
}

func Test_OnUnknownCommand_ShouldAnswerWithHelpMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	sender := mocks.NewMockMessageSender(ctrl)
	// Ожидаем ответ, что такая команда неизвестна.
	sender.EXPECT().SendMessage(txtUnknownCommand, int64(123))

	model := New(context.Background(), sender, nil, nil, nil, nil)
	err := model.IncomingMessage(Message{
		Text:   "some test text",
		UserID: 123,
	})

	assert.NoError(t, err)
}

func Test_parseLineRec_ShouldFillFields(t *testing.T) {
	line := "2022-09-20 1500 Кино"
	userDataRec, err := parseLineRec(line)

	assert.NoError(t, err)
	assert.Equal(t,
		types.UserDataRecord{
			Category: "Кино",
			Sum:      150000,
			Period:   time.Date(2022, 9, 20, 0, 0, 0, 0, time.UTC),
		},
		userDataRec,
	)
}

func Test_parseLineRec_ShouldFillFields_WhenSumIsFloat(t *testing.T) {
	line := "2022-07-12 350.50 Продукты, еда"
	userDataRec, err := parseLineRec(line)

	assert.NoError(t, err)
	assert.Equal(t,
		types.UserDataRecord{
			Category: "Продукты, еда",
			Sum:      35050,
			Period:   time.Date(2022, 7, 12, 0, 0, 0, 0, time.UTC),
		},
		userDataRec,
	)
}

func Test_parseLineRec_ShouldReturnError_WhenNoSum(t *testing.T) {
	line := "2022-04-10 Кошка"
	_, err := parseLineRec(line)

	assert.Error(t, err)
}

func Test_parseLineRec_ShouldReturnError_WhenBadDate(t *testing.T) {
	line := "2022-22-05 150 Еда"
	_, err := parseLineRec(line)

	assert.Error(t, err)
}
