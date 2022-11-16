package net_http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type HttpClient[T any] struct {
	HttpClient http.Client
}

func New[T any]() *HttpClient[T] {
	var netClient = http.Client{
		Timeout: time.Second * 5,
	}
	clt := &HttpClient[T]{
		HttpClient: netClient,
	}
	return clt
}

// Отправка запроса по указанному URL, получение JSON и запись в указанную структуру.
func (clt *HttpClient[T]) GetJsonByURL(ctx context.Context, url string, jsonStruct *T) error {

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := clt.HttpClient.Do(request)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		return err
	}

	jsonErr := json.Unmarshal(body, jsonStruct)

	if jsonErr != nil {
		return err
	}

	return nil
}
