// Package timeutils Хелпер для операций с датами и временем
package timeutils

import "time"

// BeginOfMonth Функция возвращает момент начала месяца указанной даты.
// Например, при t = "16.10.2022 15:22:30" функция вернет дату "01.10.2022 00:00:00"
func BeginOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// BeginOfNextMonth Функция возвращает момент начала следующего месяца указанной даты.
// Например, при t = "16.10.2022 15:22:30" функция вернет дату "01.11.2022 00:00:00"
func BeginOfNextMonth(t time.Time) time.Time {
	m := t.Month() + 1
	y := t.Year()
	if m > 12 {
		m = 1
		y++
	}
	return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
}
