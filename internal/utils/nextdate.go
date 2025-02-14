package utils

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// NextDate вычисляет следующую дату выполнения задачи.
// Параметры:
//
//	now     — время, от которого ищется ближайшая дата (ожидается, что задача уже выполнена).
//	dateStr — исходная дата задачи в формате "20060102" (начало повторений).
//	repeat  — правило повторения: либо "d <число>" (интервал в днях, максимум 400), либо "y" (ежегодно).
//
// Функция всегда возвращает дату, которая строго больше now.
func NextDate(now time.Time, dateStr string, repeat string) (string, error) {
	// Используем явную временную зону UTC
	loc := time.UTC

	if strings.TrimSpace(repeat) == "" {
		return "", errors.New("поле repeat пустое")
	}

	// Парсим базовую дату (без учёта монотонного времени)
	baseDate, err := time.ParseInLocation("20060102", dateStr, loc)
	if err != nil {
		return "", err
	}
	// Приводим now к той же зоне
	now = now.In(loc)

	// В зависимости от типа правила повторения
	if strings.HasPrefix(repeat, "d ") {
		// Разбор правила вида "d <число>"
		parts := strings.SplitN(repeat, " ", 2)
		if len(parts) != 2 {
			return "", errors.New("некорректный формат: не указан интервал дней")
		}
		interval, err := strconv.Atoi(parts[1])
		if err != nil || interval <= 0 {
			return "", errors.New("некорректный интервал дней")
		}
		if interval > 400 {
			return "", errors.New("интервал дней превышает допустимый предел (400)")
		}
		// Всегда сдвигаем задачу хотя бы один раз
		candidate := baseDate.AddDate(0, 0, interval)
		// Если после первого сдвига дата всё ещё не больше now, прибавляем повторно
		for !candidate.After(now) {
			candidate = candidate.AddDate(0, 0, interval)
		}
		return candidate.Format("20060102"), nil

	} else if repeat == "y" {
		// Годовое повторение
		candidate := baseDate.AddDate(1, 0, 0)
		// Если исходная дата — 29 февраля, а после прибавления года получилась 28 февраля,
		// корректируем до 1 марта
		if baseDate.Month() == time.February && baseDate.Day() == 29 &&
			candidate.Month() == time.February && candidate.Day() == 28 {
			candidate = candidate.AddDate(0, 0, 1)
		}
		// Если полученная дата не больше now, прибавляем год повторно
		for !candidate.After(now) {
			candidate = candidate.AddDate(1, 0, 0)
			if baseDate.Month() == time.February && baseDate.Day() == 29 &&
				candidate.Month() == time.February && candidate.Day() == 28 {
				candidate = candidate.AddDate(0, 0, 1)
			}
		}
		return candidate.Format("20060102"), nil

	} else {
		return "", errors.New("неподдерживаемый формат repeat")
	}
}
