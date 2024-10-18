package utils

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

const layout = "20060102"

func NextDate(now time.Time, date string, repeat string, status string) (string, error) {

	if date == "" {
		return "", nil
	}

	startDate, err := time.Parse(layout, date)
	if err != nil {
		return "", nil
	}

	if repeat == "" {
		if startDate.After(now) {
			return startDate.Format(layout), nil
		}
		return "", nil
	}

	// проверка на ежедневный интервал
	if strings.HasPrefix(repeat, "d ") {
		daysStr := strings.TrimPrefix(repeat, "d ")
		days, err := strconv.Atoi(daysStr)

		if err != nil || days < 1 || days > 400 {
			return "", nil
		}

		if status != "done" {
			if isSameDate(startDate, now) {
				return startDate.Format("20060102"), nil
			}
		}

		nextDate := startDate.AddDate(0, 0, days)

		for !nextDate.After(now) {
			nextDate = nextDate.AddDate(0, 0, days)
		}

		return nextDate.Format(layout), nil
	}

	// проверка на ежегодный интервал
	if repeat == "y" {
		nextDate := startDate.AddDate(1, 0, 0)

		// обработка 29 февраля
		if startDate.Month() == time.February && startDate.Day() == 29 {
			// проверка на високосный год следующего года
			if nextDate.Month() != time.February || nextDate.Day() != 29 {
				// переход на 1 марта, если след. год не високосный
				nextDate = time.Date(nextDate.Year(), time.March, 1, 0, 0, 0, 0, nextDate.Location())
			}
		}

		if nextDate.Before(now) {
			for !nextDate.After(now) {
				nextDate = nextDate.AddDate(1, 0, 0)
			}
		}
		return nextDate.Format(layout), nil
	}

	return "", errors.New("неподдерживаемый формат повтора")
}

func isSameDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}
